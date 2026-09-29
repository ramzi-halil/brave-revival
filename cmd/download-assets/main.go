package main

import (
	"context"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type asset struct {
	hash   string
	size   int64
	url    string
	target string
}

type byteSize int64

func (b byteSize) LogValue() slog.Value {
	if b > -1024 && b < 1024 {
		return slog.StringValue(fmt.Sprintf("%d B", b))
	}

	value := float64(b)
	unit := "B"
	for _, next := range []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"} {
		if value > -1024 && value < 1024 {
			break
		}
		value /= 1024
		unit = next
	}
	precision := 2
	if value <= -100 || value >= 100 {
		precision = 0
	} else if value <= -10 || value >= 10 {
		precision = 1
	}
	return slog.StringValue(strconv.FormatFloat(value, 'f', precision, 64) + " " + unit)
}

type counters struct {
	downloadedFiles atomic.Int64
	downloadedBytes atomic.Int64
	skippedFiles    atomic.Int64
	skippedBytes    atomic.Int64
	failedFiles     atomic.Int64
}

type errorLimiter struct {
	mu         sync.Mutex
	times      []time.Time
	suppressed int64
}

func (l *errorLimiter) log(a asset, err error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for len(l.times) > 0 && now.Sub(l.times[0]) >= 10*time.Second {
		l.times = l.times[1:]
	}
	if len(l.times) == 3 {
		l.suppressed++
		return
	}
	l.times = append(l.times, now)
	slog.Error("asset download failed", "hash", a.hash, "url", a.url, "error", err)
}

func loadAssets(csvPath, output, platform, baseURL, prefix string) ([]asset, int64, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return nil, 0, fmt.Errorf("read CSV header: %w", err)
	}
	if !slices.Equal(header, []string{"id", "ios", "android", "ios_size", "android_size"}) {
		return nil, 0, fmt.Errorf("unexpected resources CSV header: %v", header)
	}

	platformColumn, sizeColumn := 1, 3
	if platform == "android" {
		platformColumn, sizeColumn = 2, 4
	}

	var assets []asset
	var totalBytes int64
	seen := make(map[string]int64)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read resources CSV: %w", err)
		}
		if !strings.HasPrefix(record[0], prefix) {
			continue
		}

		hash := record[platformColumn]
		decoded, err := hex.DecodeString(hash)
		if err != nil || len(decoded) != md5.Size {
			return nil, 0, fmt.Errorf("invalid hash %q for ID %s", hash, record[0])
		}
		size, err := strconv.ParseInt(record[sizeColumn], 10, 64)
		if err != nil || size < 0 {
			return nil, 0, fmt.Errorf("invalid size %q for ID %s", record[sizeColumn], record[0])
		}
		if previous, ok := seen[hash]; ok {
			if previous != size {
				return nil, 0, fmt.Errorf("conflicting sizes for hash %s", hash)
			}
			continue
		}
		seen[hash] = size

		location := platform
		if record[1] == record[2] {
			location = "share"
		}
		assetURL, err := url.JoinPath(baseURL, location, hash)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid asset URL: %w", err)
		}
		assets = append(assets, asset{
			hash:   hash,
			size:   size,
			url:    assetURL,
			target: filepath.Join(output, hash[:2], hash+".unity3d"),
		})
		totalBytes += size
	}
	return assets, totalBytes, nil
}

func downloadAsset(ctx context.Context, client *http.Client, a asset) (bool, error) {
	if info, err := os.Stat(a.target); err == nil {
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("target is not a regular file: %s", a.target)
		}
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("check target: %w", err)
	}

	dir := filepath.Dir(a.target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, fmt.Errorf("create output directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+a.hash+"-*.part")
	if err != nil {
		return false, fmt.Errorf("create temporary file: %w", err)
	}
	defer os.Remove(tmp.Name())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.url, nil)
	if err != nil {
		tmp.Close()
		return false, fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		tmp.Close()
		return false, fmt.Errorf("GET asset: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		tmp.Close()
		return false, fmt.Errorf("GET asset: HTTP %s", resp.Status)
	}

	digest := md5.New()
	size, copyErr := io.Copy(io.MultiWriter(tmp, digest), resp.Body)
	closeErr := tmp.Close()
	if copyErr != nil {
		return false, fmt.Errorf("read asset: %w", copyErr)
	}
	if closeErr != nil {
		return false, fmt.Errorf("close temporary file: %w", closeErr)
	}
	if size != a.size {
		return false, fmt.Errorf("size mismatch: downloaded %d, expected %d", size, a.size)
	}
	if actual := hex.EncodeToString(digest.Sum(nil)); !strings.EqualFold(actual, a.hash) {
		return false, fmt.Errorf("MD5 mismatch: downloaded %s, expected %s", actual, a.hash)
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return false, fmt.Errorf("set asset permissions: %w", err)
	}

	// A hard link publishes the verified file without replacing a file from another run.
	if err := os.Link(tmp.Name(), a.target); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, fmt.Errorf("publish asset: %w", err)
	}
	return true, nil
}

func reportProgress(c *counters, totalFiles int, totalBytes int64, started time.Time) {
	downloadedFiles := c.downloadedFiles.Load()
	downloadedBytes := c.downloadedBytes.Load()
	completedBytes := downloadedBytes + c.skippedBytes.Load()
	percent := 100.0
	if totalBytes > 0 {
		percent = 100 * float64(completedBytes) / float64(totalBytes)
	}
	eta := "unknown"
	if completedBytes >= totalBytes {
		eta = "0s"
	} else if downloadedBytes > 0 {
		rate := float64(downloadedBytes) / time.Since(started).Seconds()
		eta = (time.Duration(float64(totalBytes-completedBytes) / rate * float64(time.Second))).Round(time.Second).String()
	}
	slog.Info("download progress", "downloaded_files", downloadedFiles, "downloaded_bytes", byteSize(downloadedBytes),
		"skipped_files", c.skippedFiles.Load(), "total_files", totalFiles, "total_bytes", byteSize(totalBytes),
		"progress_percent", fmt.Sprintf("%.1f", percent), "eta", eta, "failed_files", c.failedFiles.Load())
}

func run(ctx context.Context, csvPath, output, platform, baseURL, prefix string, concurrency int, dryRun bool) (int64, error) {
	if platform != "android" && platform != "ios" {
		return 0, fmt.Errorf("invalid platform %q: use android or ios", platform)
	}
	if concurrency < 1 {
		return 0, fmt.Errorf("concurrency must be positive")
	}
	slog.Info("starting download", "resources_csv", csvPath, "output_dir", output, "platform", platform,
		"base_url", baseURL, "prefix", prefix)
	assets, totalBytes, err := loadAssets(csvPath, output, platform, baseURL, prefix)
	if err != nil {
		return 0, err
	}
	if dryRun {
		slog.Info("dry run", "total_files", len(assets), "total_bytes", byteSize(totalBytes))
		return 0, nil
	}

	started := time.Now()
	client := &http.Client{Timeout: 30 * time.Minute}
	c := &counters{}
	limiter := &errorLimiter{}
	jobs := make(chan asset)
	var workers sync.WaitGroup
	for range concurrency {
		workers.Go(func() {
			for a := range jobs {
				installed, err := downloadAsset(ctx, client, a)
				if err != nil {
					c.failedFiles.Add(1)
					limiter.log(a, err)
				} else if installed {
					c.downloadedFiles.Add(1)
					c.downloadedBytes.Add(a.size)
				} else {
					c.skippedFiles.Add(1)
					c.skippedBytes.Add(a.size)
				}
			}
		})
	}

	stopProgress := make(chan struct{})
	progressDone := make(chan struct{})
	go func() {
		defer close(progressDone)
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				reportProgress(c, len(assets), totalBytes, started)
			case <-stopProgress:
				return
			}
		}
	}()

	for _, a := range assets {
		jobs <- a
	}
	close(jobs)
	workers.Wait()
	close(stopProgress)
	<-progressDone

	slog.Info("download finished", "downloaded_files", c.downloadedFiles.Load(),
		"downloaded_bytes", byteSize(c.downloadedBytes.Load()), "skipped_files", c.skippedFiles.Load(),
		"failed_files", c.failedFiles.Load(), "suppressed_errors", limiter.suppressed,
		"total_files", len(assets), "total_bytes", byteSize(totalBytes), "elapsed", time.Since(started).Round(time.Millisecond))
	return c.failedFiles.Load(), nil
}

func main() {
	res := flag.String("i", "./db/master/resources.csv", "path to resources.csv")
	output := flag.String("o", "./raw", "output directory")
	platform := flag.String("p", "", "platform to download assets for (android or ios)")
	baseURL := flag.String("u", "https://assets-production.enish-games.com/crow/Assets", "base URL of the server to download assets from")
	concurrency := flag.Int("j", 2, "number of parallel download threads")
	prefix := flag.String("f", "", "only download assets whose ID starts with this prefix")
	dryRun := flag.Bool("n", false, "dry run, do not download anything, just check the total size")
	flag.Parse()

	failures, err := run(context.Background(), *res, *output, *platform, *baseURL, *prefix, *concurrency, *dryRun)
	if err != nil {
		slog.Error("download could not start", "error", err)
		os.Exit(1)
	}
	if failures > 0 {
		os.Exit(1)
	}
}
