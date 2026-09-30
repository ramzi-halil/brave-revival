package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"io"
	"math/big"
	"os"
	"path"
	"strings"
	"time"

	"github.com/agusibrahim/apksig-go/pkg/algo"
	"github.com/agusibrahim/apksig-go/pkg/apkverifier"
	"github.com/agusibrahim/apksig-go/pkg/apkwriter"
	"github.com/agusibrahim/apksig-go/pkg/axml"
	"github.com/agusibrahim/apksig-go/pkg/datasource"
	"github.com/agusibrahim/apksig-go/pkg/signer"
)

func loadSigner(file string) (*signer.SignerConfig, error) {
	data, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
		if err != nil {
			return nil, err
		}
		now := time.Now()
		template := &x509.Certificate{
			SerialNumber: serial, Subject: pkix.Name{CommonName: "Brave Revival APK"},
			NotBefore: now.Add(-time.Hour), NotAfter: now.AddDate(27, 0, 0),
			KeyUsage: x509.KeyUsageDigitalSignature,
		}
		der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if err != nil {
			return nil, err
		}
		data = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
		data = append(data, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})...)
		f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return nil, err
		}
		_, writeErr := f.Write(data)
		closeErr := f.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	pair, err := tls.X509KeyPair(data, data)
	if err != nil {
		return nil, err
	}
	if _, ok := pair.PrivateKey.(*rsa.PrivateKey); !ok {
		return nil, fmt.Errorf("signing PEM must contain an RSA private key")
	}
	cert, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, err
	}
	algorithm, _ := algo.ByID(algo.SigRSAPKCS1SHA256)
	return &signer.SignerConfig{PrivateKey: pair.PrivateKey, Certs: []*x509.Certificate{cert}, Algorithms: []algo.Algorithm{algorithm}}, nil
}

func readEntry(f *zip.File) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func oldSignature(name string) bool {
	name = strings.ToUpper(name)
	if path.Dir(name) != "META-INF" {
		return false
	}
	base := path.Base(name)
	return base == "MANIFEST.MF" || strings.HasPrefix(base, "SIG-") ||
		strings.HasSuffix(base, ".SF") || strings.HasSuffix(base, ".RSA") ||
		strings.HasSuffix(base, ".DSA") || strings.HasSuffix(base, ".EC")
}

// CreateRaw preserves compressed payloads; padding belongs in the local header,
// so alignment is applied before signing and also covers stored native libraries.
func writeRaw(w *zip.Writer, out *bytes.Buffer, h zip.FileHeader, r io.Reader) error {
	// Sizes are known. Omitting descriptors also keeps the next header's offset
	// final before CreateRaw closes the preceding entry.
	h.Flags &^= 8
	if err := w.Flush(); err != nil {
		return err
	}
	// TODO: Reuse apksig-go's aligner once https://github.com/agusibrahim/apksig-go/pull/1
	// is available in a release; v1.1.0 only aligns to 4 bytes and can discard
	// existing extra fields, whereas stored .so entries need page alignment.
	if h.Method == zip.Store && !strings.HasSuffix(h.Name, "/") {
		alignment := 4
		if strings.HasPrefix(h.Name, "lib/") && strings.HasSuffix(h.Name, ".so") {
			alignment = 16384
		}
		offset := out.Len() + 30 + len(h.Name) + len(h.Extra)
		if offset%alignment != 0 {
			pad := (alignment - (offset+4)%alignment) % alignment
			extra := make([]byte, 4+pad)
			put16(extra, 0, 0xffff)
			put16(extra, 2, uint16(pad))
			h.Extra = append(bytes.Clone(h.Extra), extra...)
		}
		if len(h.Extra) > 65535 {
			return fmt.Errorf("extra field too long for %s", h.Name)
		}
	}
	fw, err := w.CreateRaw(&h)
	if err != nil {
		return err
	}
	_, err = io.Copy(fw, r)
	return err
}

func storedHeader(name string, data []byte) zip.FileHeader {
	return zip.FileHeader{Name: name, Method: zip.Store, CRC32: crc32.ChecksumIEEE(data), CompressedSize64: uint64(len(data)), UncompressedSize64: uint64(len(data))}
}

func patchAPK(data []byte, base bool, cfg *signer.SignerConfig) ([]byte, error) {
	a, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	var manifest, resources []byte
	for _, f := range a.File {
		if f.Name == "AndroidManifest.xml" {
			manifest, err = readEntry(f)
		}
		if base && f.Name == "resources.arsc" {
			resources, err = readEntry(f)
		}
		if err != nil {
			return nil, err
		}
	}
	if manifest == nil {
		return nil, fmt.Errorf("missing AndroidManifest.xml")
	}
	if base {
		minSDK, err := axml.MinSdk(manifest)
		if err != nil {
			return nil, err
		}
		if minSDK < 24 {
			return nil, fmt.Errorf("v2-only signing requires minSdkVersion >= 24, got %d", minSDK)
		}
		var id uint32
		resources, id, err = addResource(resources)
		if err != nil {
			return nil, fmt.Errorf("patch resources: %w", err)
		}
		manifest, err = patchManifest(manifest, id)
		if err != nil {
			return nil, fmt.Errorf("patch manifest: %w", err)
		}
	}
	var unsigned bytes.Buffer
	w := zip.NewWriter(&unsigned)
	for _, f := range a.File {
		if oldSignature(f.Name) {
			continue
		}
		var replacement []byte
		if base && f.Name == configPath {
			return nil, fmt.Errorf("config path already exists")
		}
		if base && f.Name == "AndroidManifest.xml" {
			replacement = manifest
		}
		if base && f.Name == "resources.arsc" {
			replacement = resources
		}
		if replacement != nil {
			if err := writeRaw(w, &unsigned, storedHeader(f.Name, replacement), bytes.NewReader(replacement)); err != nil {
				return nil, err
			}
		} else {
			r, err := f.OpenRaw()
			if err != nil {
				return nil, err
			}
			if err := writeRaw(w, &unsigned, f.FileHeader, r); err != nil {
				return nil, err
			}
		}
	}
	if base {
		config := networkConfig()
		if err := writeRaw(w, &unsigned, storedHeader(configPath, config), bytes.NewReader(config)); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	var signed bytes.Buffer
	sw := apkwriter.SignedAPKWriter{Src: datasource.NewBytes(unsigned.Bytes()), Signers: []*signer.SignerConfig{cfg}}
	if err := sw.Write(&signed); err != nil {
		return nil, err
	}
	result, err := apkverifier.Verify(datasource.NewBytes(signed.Bytes()), 24, 35)
	if err != nil {
		return nil, err
	}
	// Split manifests can omit uses-sdk and inherit the base's SDK range.
	if !result.V2Verified || len(result.SignerCerts) != 1 || !bytes.Equal(result.SignerCerts[0], cfg.Certs[0].Raw) {
		return nil, fmt.Errorf("v2 signature verification failed: %v", result.Errors)
	}
	return signed.Bytes(), nil
}

func patchXAPK(input, output string, cfg *signer.SignerConfig) (err error) {
	z, err := zip.OpenReader(input)
	if err != nil {
		return err
	}
	defer z.Close()
	var meta map[string]json.RawMessage
	for _, f := range z.File {
		if f.Name != "manifest.json" {
			continue
		}
		data, e := readEntry(f)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(data, &meta); e != nil {
			return e
		}
	}
	var splits []struct {
		File string `json:"file"`
		ID   string `json:"id"`
	}
	if err := json.Unmarshal(meta["split_apks"], &splits); err != nil {
		return fmt.Errorf("read split_apks: %w", err)
	}
	base := ""
	for _, s := range splits {
		if s.ID == "base" {
			if base != "" {
				return fmt.Errorf("multiple base APKs")
			}
			base = s.File
		}
	}
	if base == "" {
		return fmt.Errorf("manifest.json has no base split")
	}
	seen := make(map[string]bool)
	for _, f := range z.File {
		if seen[f.Name] {
			return fmt.Errorf("duplicate XAPK entry %s", f.Name)
		}
		seen[f.Name] = true
	}
	for _, s := range splits {
		if !seen[s.File] || !strings.HasSuffix(s.File, ".apk") {
			return fmt.Errorf("missing/invalid APK %s", s.File)
		}
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			f.Close()
			os.Remove(output)
		}
	}()
	w := zip.NewWriter(f)
	var total uint64
	for _, entry := range z.File {
		if entry.Name == "manifest.json" {
			continue
		}
		if !strings.HasSuffix(entry.Name, ".apk") {
			if err := w.Copy(entry); err != nil {
				return err
			}
			continue
		}
		fmt.Fprintf(os.Stderr, "Patching/signing %s\n", entry.Name)
		data, err := readEntry(entry)
		if err != nil {
			return err
		}
		data, err = patchAPK(data, entry.Name == base, cfg)
		if err != nil {
			return fmt.Errorf("%s: %w", entry.Name, err)
		}
		total += uint64(len(data))
		fw, err := w.CreateHeader(&zip.FileHeader{Name: entry.Name, Method: zip.Store})
		if err != nil {
			return err
		}
		if _, err := fw.Write(data); err != nil {
			return err
		}
	}
	meta["total_size"], err = json.Marshal(total)
	if err != nil {
		return err
	}
	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	fw, err := w.CreateHeader(&zip.FileHeader{Name: "manifest.json", Method: zip.Store})
	if err != nil {
		return err
	}
	if _, err := fw.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return f.Close()
}

func main() {
	output := flag.String("o", "patched.xapk", "output XAPK (must not exist)")
	key := flag.String("key", "apk-signing.pem", "persistent signing key/certificate PEM (created if absent)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "Usage: patch-xapk [-o patched.xapk] [-key apk-signing.pem] input.xapk")
		os.Exit(2)
	}
	cfg, err := loadSigner(*key)
	if err == nil {
		err = patchXAPK(flag.Arg(0), *output, cfg)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %s\nSigner SHA-256: %x\n", *output, sha256.Sum256(cfg.Certs[0].Raw))
}
