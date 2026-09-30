package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/agusibrahim/apksig-go/pkg/apkverifier"
	"github.com/agusibrahim/apksig-go/pkg/datasource"
	"github.com/shogo82148/androidbinary"
)

func testPool(t *testing.T, names ...string) []byte {
	t.Helper()
	p := chunk(1, 28, 28)
	put32(p, 16, 256)
	put32(p, 20, 28)
	for _, name := range names {
		var err error
		p, _, err = appendString(p, name)
		if err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func testResources(t *testing.T) []byte {
	t.Helper()
	types, keys := testPool(t, "xml"), testPool(t, "original")
	pkg := chunk(0x200, 288, 288)
	put32(pkg, 8, 0x7f)
	put32(pkg, 268, 288)
	put32(pkg, 276, uint32(288+len(types)))
	spec := chunk(0x202, 16, 20)
	spec[8] = 1
	put32(spec, 12, 1)
	typ := chunk(0x201, 84, 104)
	typ[8] = 1
	put32(typ, 12, 1)
	put32(typ, 16, 88)
	put32(typ, 20, 64)
	put16(typ, 88, 8)
	put16(typ, 96, 8)
	typ[99] = 3
	pkg = assemble(pkg, [][]byte{types, keys, spec, typ})
	table := chunk(2, 12, 12)
	put32(table, 8, 1)
	return assemble(table, [][]byte{testPool(t, "res/xml/original.xml"), pkg})
}

func testManifest(t *testing.T) []byte {
	t.Helper()
	names := []string{"hasCode", "minSdkVersion", "targetSdkVersion", "name", "exported", "android", androidNS, "manifest", "package", "com.braverevival.netsecprobe", "uses-sdk", "application", "activity", "android.app.Activity"}
	p := testPool(t, names...)
	m := chunk(0x180, 8, 28)
	for i, id := range []uint32{0x0101000c, 0x0101020c, 0x01010270, 0x01010003, 0x01010010} {
		put32(m, 8+4*i, id)
	}
	parts := [][]byte{p, m}
	namespace := func(kind uint16) {
		c := chunk(kind, 16, 24)
		put32(c, 8, 1)
		put32(c, 12, noEntry)
		put32(c, 16, 5)
		put32(c, 20, 6)
		parts = append(parts, c)
	}
	attr := func(ns, name uint32, typ byte, value uint32) []byte {
		b := make([]byte, 20)
		put32(b, 0, ns)
		put32(b, 4, name)
		put32(b, 8, noEntry)
		put16(b, 12, 8)
		b[15] = typ
		put32(b, 16, value)
		if typ == 3 {
			put32(b, 8, value)
		}
		return b
	}
	start := func(name uint32, attrs ...[]byte) {
		c := chunk(0x102, 16, 36)
		put32(c, 8, 1)
		put32(c, 12, noEntry)
		put32(c, 16, noEntry)
		put32(c, 20, name)
		put16(c, 24, 20)
		put16(c, 26, 20)
		put16(c, 28, uint16(len(attrs)))
		for _, a := range attrs {
			c = append(c, a...)
		}
		put32(c, 4, uint32(len(c)))
		parts = append(parts, c)
	}
	end := func(name uint32) {
		c := chunk(0x103, 16, 24)
		put32(c, 8, 1)
		put32(c, 12, noEntry)
		put32(c, 16, noEntry)
		put32(c, 20, name)
		parts = append(parts, c)
	}
	namespace(0x100)
	start(7, attr(noEntry, 8, 3, 9))
	start(10, attr(6, 1, 0x10, 24), attr(6, 2, 0x10, 28))
	end(10)
	start(11, attr(6, 0, 0x12, 0))
	start(12, attr(6, 3, 3, 13), attr(6, 4, 0x12, noEntry))
	end(12)
	end(11)
	end(7)
	namespace(0x101)
	return assemble(chunk(3, 8, 8), parts)
}

func decodedTokens(t *testing.T, b []byte, stripConfig bool) []xml.Token {
	t.Helper()
	f, err := androidbinary.NewXMLFile(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	d := xml.NewDecoder(f.Reader())
	var out []xml.Token
	for {
		token, err := d.Token()
		if err == io.EOF {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		if el, ok := token.(xml.StartElement); ok && stripConfig {
			el.Attr = slices.DeleteFunc(el.Attr, func(a xml.Attr) bool { return a.Name.Space == androidNS && a.Name.Local == "networkSecurityConfig" })
			token = el
		}
		out = append(out, xml.CopyToken(token))
	}
}

func verifyResources(t *testing.T, original, patched []byte, id uint32) {
	t.Helper()
	a, err := androidbinary.NewTableFile(bytes.NewReader(original))
	if err != nil {
		t.Fatal(err)
	}
	b, err := androidbinary.NewTableFile(bytes.NewReader(patched))
	if err != nil {
		t.Fatal(err)
	}
	value, err := b.GetResource(androidbinary.ResID(id), nil)
	if err != nil || value != configPath {
		t.Fatalf("new resource = %v, %v", value, err)
	}
	parts, err := children(original, 2)
	if err != nil {
		t.Fatal(err)
	}
	pc, err := children(parts[1], 0x200)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range pc {
		if u16(c, 0) != 0x202 {
			continue
		}
		for i := uint32(0); i < u32(c, 12); i++ {
			resID := androidbinary.ResID(u32(parts[1], 8)<<24 | uint32(c[8])<<16 | i)
			av, ae := a.GetResource(resID, nil)
			bv, be := b.GetResource(resID, nil)
			if (ae == nil) != (be == nil) || !reflect.DeepEqual(av, bv) {
				t.Fatalf("existing resource %s changed: %v / %v", resID, av, bv)
			}
		}
	}
}

func TestBinaryPatch(t *testing.T) {
	original := testResources(t)
	resources, id, err := addResource(original)
	if err != nil {
		t.Fatal(err)
	}
	if id != 0x7f010001 {
		t.Fatalf("unexpected ID %#x", id)
	}
	verifyResources(t, original, resources, id)
	m := testManifest(t)
	patched, err := patchManifest(m, id)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decodedTokens(t, m, false), decodedTokens(t, patched, true)) {
		t.Fatal("unrelated manifest content changed")
	}
	if _, err := patchManifest(patched, id); err == nil {
		t.Fatal("accepted existing policy")
	}
	if _, _, err := addResource(resources); err == nil {
		t.Fatal("accepted patched resources")
	}
	f, err := androidbinary.NewXMLFile(bytes.NewReader(networkConfig()))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		XMLName xml.Name `xml:"network-security-config"`
		Anchors []struct {
			Src string `xml:"src,attr"`
		} `xml:"base-config>trust-anchors>certificates"`
	}
	if err := xml.NewDecoder(f.Reader()).Decode(&config); err != nil {
		t.Fatal(err)
	}
	if len(config.Anchors) != 2 || config.Anchors[0].Src != "user" || config.Anchors[1].Src != "system" {
		t.Fatalf("unexpected trust anchors: %+v", config)
	}
}

func TestAPKSigningAndAlignment(t *testing.T) {
	key := filepath.Join(t.TempDir(), "signing.pem")
	cfg, err := loadSigner(key)
	if err != nil {
		t.Fatal(err)
	}
	again, err := loadSigner(key)
	if err != nil || !bytes.Equal(cfg.Certs[0].Raw, again.Certs[0].Raw) {
		t.Fatal("signer identity changed")
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, item := range []struct {
		name string
		data []byte
	}{
		{"AndroidManifest.xml", testManifest(t)}, {"resources.arsc", testResources(t)},
		{"before-native", []byte("entry with a data descriptor")},
		{"lib/arm64-v8a/probe.so", []byte("stored native library")},
		{"lib/arm64-v8a/second.so", []byte("another stored native library")}, {"META-INF/OLD.RSA", []byte("stale signature")},
	} {
		fw, err := w.CreateHeader(&zip.FileHeader{Name: item.name, Method: zip.Store})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(item.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	signed, err := patchAPK(buf.Bytes(), true, cfg)
	if err != nil {
		t.Fatal(err)
	}
	a, err := zip.NewReader(bytes.NewReader(signed), int64(len(signed)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range a.File {
		if oldSignature(f.Name) {
			t.Fatalf("stale signature %s", f.Name)
		}
		if f.Method == zip.Store {
			off, err := f.DataOffset()
			if err != nil {
				t.Fatal(err)
			}
			alignment := int64(4)
			if f.Name == "lib/arm64-v8a/probe.so" || f.Name == "lib/arm64-v8a/second.so" {
				alignment = 16384
			}
			if off%alignment != 0 {
				t.Fatalf("misaligned %s at %d", f.Name, off)
			}
		}
		if _, err := readEntry(f); err != nil {
			t.Fatal(err)
		}
	}
	result, err := apkverifier.Verify(datasource.NewBytes(signed), 24, 35)
	if err != nil || !result.Verified || !result.V2Verified {
		t.Fatalf("verify: %+v, %v", result, err)
	}
	signed[50] ^= 1
	result, err = apkverifier.Verify(datasource.NewBytes(signed), 24, 35)
	if err == nil && result.V2Verified {
		t.Fatal("tampered APK passed verification")
	}
}

func TestSampleExperiment(t *testing.T) {
	input := os.Getenv("PATCH_XAPK_SAMPLE")
	if input == "" {
		t.Skip("set PATCH_XAPK_SAMPLE to run the full sample experiment")
	}
	dir := t.TempDir()
	cfg, err := loadSigner(filepath.Join(dir, "signing.pem"))
	if err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "patched.xapk")
	if err := patchXAPK(input, output, cfg); err != nil {
		t.Fatal(err)
	}
	original, err := zip.OpenReader(input)
	if err != nil {
		t.Fatal(err)
	}
	defer original.Close()
	patched, err := zip.OpenReader(output)
	if err != nil {
		t.Fatal(err)
	}
	defer patched.Close()
	if len(original.File) != len(patched.File) {
		t.Fatal("outer entries changed")
	}
	var total uint64
	for _, old := range original.File {
		var next *zip.File
		for _, f := range patched.File {
			if f.Name == old.Name {
				next = f
			}
		}
		if next == nil {
			t.Fatalf("lost %s", old.Name)
		}
		if old.Name == "manifest.json" {
			continue
		}
		ob, err := readEntry(old)
		if err != nil {
			t.Fatal(err)
		}
		nb, err := readEntry(next)
		if err != nil {
			t.Fatal(err)
		}
		a, err := zip.NewReader(bytes.NewReader(ob), int64(len(ob)))
		if err != nil {
			t.Fatal(err)
		}
		b, err := zip.NewReader(bytes.NewReader(nb), int64(len(nb)))
		if err != nil {
			t.Fatal(err)
		}
		total += uint64(len(nb))
		result, err := apkverifier.Verify(datasource.NewBytes(nb), 24, 35)
		if err != nil || !result.V2Verified || !bytes.Equal(result.SignerCerts[0], cfg.Certs[0].Raw) {
			t.Fatalf("signature %s: %+v, %v", old.Name, result, err)
		}
		for _, f := range b.File {
			if f.Method == zip.Store {
				off, err := f.DataOffset()
				if err != nil || off%4 != 0 {
					t.Fatalf("alignment %s: %d, %v", f.Name, off, err)
				}
			}
		}
		for _, f := range a.File {
			if oldSignature(f.Name) {
				continue
			}
			var newFile *zip.File
			for _, nf := range b.File {
				if nf.Name == f.Name {
					newFile = nf
				}
			}
			if newFile == nil {
				t.Fatalf("lost %s", f.Name)
			}
			ov, err := readEntry(f)
			if err != nil {
				t.Fatal(err)
			}
			nv, err := readEntry(newFile)
			if err != nil {
				t.Fatal(err)
			}
			if old.Name == "jp.enish.shingekibo.apk" && f.Name == "AndroidManifest.xml" {
				if !reflect.DeepEqual(decodedTokens(t, ov, false), decodedTokens(t, nv, true)) {
					t.Fatal("unrelated manifest content changed")
				}
			} else if old.Name == "jp.enish.shingekibo.apk" && f.Name == "resources.arsc" {
				verifyResources(t, ov, nv, 0x7f130004)
			} else if !bytes.Equal(ov, nv) || f.Method != newFile.Method {
				t.Fatalf("payload/method changed: %s", f.Name)
			}
		}
	}
	for _, f := range patched.File {
		if f.Name != "manifest.json" {
			continue
		}
		data, err := readEntry(f)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}
		n, err := strconv.ParseUint(string(m["total_size"]), 10, 64)
		if err != nil || n != total {
			t.Fatalf("total_size: %d != %d", n, total)
		}
	}
}

func TestAndroidProbe(t *testing.T) {
	output := os.Getenv("PATCH_XAPK_PROBE")
	if output == "" {
		t.Skip("set PATCH_XAPK_PROBE to generate a disposable Android resource probe")
	}
	cfg, err := loadSigner(filepath.Join(t.TempDir(), "signing.pem"))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, item := range []struct {
		name string
		data []byte
	}{{"AndroidManifest.xml", testManifest(t)}, {"resources.arsc", testResources(t)}} {
		fw, err := w.Create(item.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(item.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := patchAPK(buf.Bytes(), true, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, data, 0644); err != nil {
		t.Fatal(err)
	}
}
