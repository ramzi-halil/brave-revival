package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func testHash(body string) string {
	sum := md5.Sum([]byte(body))
	return hex.EncodeToString(sum[:])
}

func TestByteSizeLogValue(t *testing.T) {
	const kib = 1024
	const mib = kib * kib
	const gib = mib * kib
	for _, test := range []struct {
		bytes byteSize
		want  string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{kib, "1.00 KiB"},
		{1536, "1.50 KiB"},
		{12*mib + 300*kib, "12.3 MiB"},
		{123 * gib, "123 GiB"},
		{-1536, "-1.50 KiB"},
	} {
		if got := test.bytes.LogValue(); got.Kind() != slog.KindString || got.String() != test.want {
			t.Errorf("byteSize(%d).LogValue() = %v, want %q", test.bytes, got, test.want)
		}
	}
}

func TestRunDownloadAndResume(t *testing.T) {
	iosBody := "ios asset"
	androidBody := "android asset"
	sharedBody := "shared asset"
	corruptBody := "good"
	wrongSizeBody := "right size"
	missingBody := "missing asset"
	existingBody := "already here"
	iosHash := testHash(iosBody)
	androidHash := testHash(androidBody)
	sharedHash := testHash(sharedBody)
	corruptHash := testHash(corruptBody)
	wrongSizeHash := testHash(wrongSizeBody)
	missingHash := testHash(missingBody)
	existingHash := testHash(existingBody)

	output := t.TempDir()
	existingPath := filepath.Join(output, existingHash[:2], existingHash+".unity3d")
	if err := os.MkdirAll(filepath.Dir(existingPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingPath, []byte(existingBody), 0o644); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	requests := make(map[string]int)
	repaired := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests[r.URL.Path]++
		fixed := repaired
		mu.Unlock()
		switch r.URL.Path {
		case "/Assets/android/" + androidHash:
			fmt.Fprint(w, androidBody)
		case "/Assets/share/" + sharedHash:
			fmt.Fprint(w, sharedBody)
		case "/Assets/android/" + corruptHash:
			if fixed {
				fmt.Fprint(w, corruptBody)
			} else {
				fmt.Fprint(w, "baad")
			}
		case "/Assets/android/" + wrongSizeHash:
			if fixed {
				fmt.Fprint(w, wrongSizeBody)
			} else {
				fmt.Fprint(w, "short")
			}
		case "/Assets/android/" + missingHash:
			if fixed {
				fmt.Fprint(w, missingBody)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	csvPath := filepath.Join(t.TempDir(), "resources.csv")
	rows := []string{
		"id,ios,android,ios_size,android_size",
		fmt.Sprintf("100,%s,%s,%d,%d", iosHash, androidHash, len(iosBody), len(androidBody)),
		fmt.Sprintf("101,%s,%s,%d,%d", sharedHash, sharedHash, len(sharedBody), len(sharedBody)),
		fmt.Sprintf("102,%s,%s,%d,%d", iosHash, androidHash, len(iosBody), len(androidBody)),
		fmt.Sprintf("103,%s,%s,%d,%d", iosHash, corruptHash, len(iosBody), len(corruptBody)),
		fmt.Sprintf("104,%s,%s,%d,%d", iosHash, wrongSizeHash, len(iosBody), len(wrongSizeBody)),
		fmt.Sprintf("105,%s,%s,%d,%d", iosHash, missingHash, len(iosBody), len(missingBody)),
		fmt.Sprintf("106,%s,%s,%d,%d", iosHash, existingHash, len(iosBody), len(existingBody)),
	}
	if err := os.WriteFile(csvPath, []byte(strings.Join(rows, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	failures, err := run(context.Background(), csvPath, output, "android", server.URL+"/Assets", "", 3, false)
	if err != nil || failures != 3 {
		t.Fatalf("first run: failures=%d, err=%v", failures, err)
	}
	for hash, body := range map[string]string{androidHash: androidBody, sharedHash: sharedBody, existingHash: existingBody} {
		path := filepath.Join(output, hash[:2], hash+".unity3d")
		contents, err := os.ReadFile(path)
		if err != nil || string(contents) != body {
			t.Errorf("asset %s: contents=%q, err=%v", hash, contents, err)
		}
	}
	for _, hash := range []string{corruptHash, wrongSizeHash, missingHash} {
		path := filepath.Join(output, hash[:2], hash+".unity3d")
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("failed asset %s was published: %v", hash, err)
		}
	}
	if parts, err := filepath.Glob(filepath.Join(output, "*", "*.part")); err != nil || len(parts) != 0 {
		t.Errorf("temporary files remain: %v, %v", parts, err)
	}

	mu.Lock()
	repaired = true
	mu.Unlock()
	failures, err = run(context.Background(), csvPath, output, "android", server.URL+"/Assets", "", 3, false)
	if err != nil || failures != 0 {
		t.Fatalf("second run: failures=%d, err=%v", failures, err)
	}
	mu.Lock()
	defer mu.Unlock()
	if requests["/Assets/android/"+androidHash] != 1 || requests["/Assets/share/"+sharedHash] != 1 {
		t.Errorf("existing assets were downloaded again: %v", requests)
	}
	if requests["/Assets/android/"+corruptHash] != 2 || requests["/Assets/android/"+wrongSizeHash] != 2 || requests["/Assets/android/"+missingHash] != 2 {
		t.Errorf("failed assets were not retried: %v", requests)
	}
	if requests["/Assets/android/"+existingHash] != 0 || requests["/Assets/ios/"+iosHash] != 0 {
		t.Errorf("unexpected asset requests: %v", requests)
	}
}

func TestErrorLimiter(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	defer slog.SetDefault(previous)

	limiter := &errorLimiter{}
	for range 5 {
		limiter.log(asset{hash: "hash"}, fmt.Errorf("failed"))
	}
	if count := strings.Count(output.String(), "asset download failed"); count != 3 {
		t.Errorf("logged %d errors, want 3", count)
	}
	if limiter.suppressed != 2 {
		t.Errorf("suppressed %d errors, want 2", limiter.suppressed)
	}
}
