package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/pierrec/lz4/v4"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	reference, err := os.ReadFile("testdata/reference.unity3d")
	require.NoError(t, err)
	referenceAsset, _ := decodeTestBundle(t, reference)
	for _, platform := range []string{"ios", "android"} {
		t.Run(platform, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "output.unity3d")
			require.NoError(t, run([]string{"-i", "testdata/input.png", "-o", output, "-p", platform, "-r", "700101012"}, &bytes.Buffer{}))
			actual, err := os.ReadFile(output)
			require.NoError(t, err)
			actualAsset, blockFlags := decodeTestBundle(t, actual)
			require.Equal(t, []uint16{3}, blockFlags)
			expected := bytes.Clone(referenceAsset)
			if platform == "android" {
				asset := bytes.LastIndex(expected, []byte(unityVersion+"\x00"))
				require.Greater(t, asset, 0)
				binary.LittleEndian.PutUint32(expected[asset+len(unityVersion)+1:], 13)
			}
			require.Equal(t, expected, actualAsset)
		})
	}
}

func decodeTestBundle(t *testing.T, data []byte) ([]byte, []uint16) {
	t.Helper()
	r := bytes.NewReader(data)
	read := func(r *bytes.Reader, v any) { require.NoError(t, binary.Read(r, binary.BigEndian, v)) }
	cstring := func(r *bytes.Reader) string {
		var value []byte
		for {
			b, err := r.ReadByte()
			require.NoError(t, err)
			if b == 0 {
				return string(value)
			}
			value = append(value, b)
		}
	}
	align := func() {
		position, err := r.Seek(0, io.SeekCurrent)
		require.NoError(t, err)
		_, err = r.Seek((position+15)&^15, io.SeekStart)
		require.NoError(t, err)
	}
	take := func(r *bytes.Reader, size uint32) []byte {
		value := make([]byte, size)
		_, err := io.ReadFull(r, value)
		require.NoError(t, err)
		return value
	}
	decompress := func(data []byte, size uint32, flags uint32) []byte {
		if flags&63 == 0 {
			require.Len(t, data, int(size))
			return data
		}
		require.Equal(t, uint32(3), flags&63)
		out := make([]byte, size)
		n, err := lz4.UncompressBlock(data, out)
		require.NoError(t, err)
		require.Equal(t, int(size), n)
		return out
	}
	require.Equal(t, "UnityFS", cstring(r))
	var version uint32
	read(r, &version)
	require.Equal(t, uint32(8), version)
	require.Equal(t, "5.x.x", cstring(r))
	require.Equal(t, unityVersion, cstring(r))
	var header struct {
		Size                            uint64
		CompressedSize, SizeInfo, Flags uint32
	}
	read(r, &header)
	require.Equal(t, uint64(len(data)), header.Size)
	require.Equal(t, uint32(0x40), header.Flags&0xc0)
	align()
	info := bytes.NewReader(decompress(take(r, header.CompressedSize), header.SizeInfo, header.Flags))
	if header.Flags&0x200 != 0 {
		align()
	}
	take(info, 16)
	var count uint32
	read(info, &count)
	type block struct {
		Size, CompressedSize uint32
		Flags                uint16
	}
	blocks := make([]block, count)
	read(info, &blocks)
	var asset []byte
	var flags []uint16
	for _, b := range blocks {
		require.LessOrEqual(t, b.Size, uint32(bundleBlockSize))
		asset = append(asset, decompress(take(r, b.CompressedSize), b.Size, uint32(b.Flags))...)
		flags = append(flags, b.Flags)
	}
	read(info, &count)
	require.Equal(t, uint32(1), count)
	var node struct {
		Offset, Size uint64
		Flags        uint32
	}
	read(info, &node)
	require.Equal(t, uint64(0), node.Offset)
	require.Equal(t, uint64(len(asset)), node.Size)
	require.Equal(t, uint32(4), node.Flags)
	require.Equal(t, "CAB-700101012r", cstring(info))
	require.Zero(t, info.Len())
	require.Zero(t, r.Len())
	return asset, flags
}

func TestBundleCompression(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 512, 128))
	copy(img.Pix, bytes.Repeat([]byte{255, 80, 30, 128}, len(img.Pix)/4))
	data, err := textureBundle(img, "700101012", 13)
	require.NoError(t, err)
	asset, flags := decodeTestBundle(t, data)
	require.Equal(t, []uint16{3, 3, 3}, flags)
	require.Equal(t, serializedAsset(img, "700101012", 13), asset)
	require.Less(t, len(data), len(asset)/10)
}

func TestIncompressibleBlock(t *testing.T) {
	random := rand.New(rand.NewPCG(123, 456))
	data := make([]byte, bundleBlockSize)
	for i := range data {
		data[i] = byte(random.Uint32())
	}
	compressor := lz4.CompressorHC{Level: lz4.Level9}
	compressed, flags, err := compressBundleBlock(&compressor, data)
	require.NoError(t, err)
	require.Zero(t, flags)
	require.Equal(t, data, compressed)
	compressed, flags, err = compressBundleBlock(&compressor, []byte{1})
	require.NoError(t, err)
	require.Zero(t, flags)
	require.Equal(t, []byte{1}, compressed)
}

func TestTexturePixels(t *testing.T) {
	img := image.NewNRGBA(image.Rect(5, 7, 8, 9))
	img.SetNRGBA(5, 7, color.NRGBA{R: 255, A: 128})
	img.SetNRGBA(5, 8, color.NRGBA{B: 255, A: 255})
	img.SetNRGBA(6, 8, color.NRGBA{R: 20, G: 40, B: 60})
	data := textureObject(img, "test")
	require.Equal(t, uint32(3), binary.LittleEndian.Uint32(data[16:]))
	require.Equal(t, uint32(2), binary.LittleEndian.Uint32(data[20:]))
	pixels := data[len(data)-16-24 : len(data)-16]
	require.Equal(t, []byte{0, 0, 255, 255, 20, 40, 60, 0}, pixels[:8])
	require.Equal(t, []byte{255, 0, 0, 128}, pixels[12:16])
}

func TestArguments(t *testing.T) {
	valid := []string{"-i", "testdata/input.png", "-o", filepath.Join(t.TempDir(), "output.unity3d"), "-p", "ios", "-r", "700101012"}
	for i := 0; i < len(valid); i += 2 {
		args := append(slices.Clone(valid[:i]), valid[i+2:]...)
		require.Error(t, run(args, &bytes.Buffer{}))
	}
	for _, replacement := range []struct {
		index int
		value string
	}{
		{1, "missing.png"}, {3, t.TempDir()}, {5, "windows"}, {7, "../texture"}, {7, "texture\\name"}, {7, "texture\x00name"}, {7, " "},
	} {
		args := slices.Clone(valid)
		args[replacement.index] = replacement.value
		require.Error(t, run(args, &bytes.Buffer{}))
	}
	require.Error(t, run(append(slices.Clone(valid), "extra"), &bytes.Buffer{}))
	require.Error(t, run([]string{"-unknown"}, &bytes.Buffer{}))
	require.ErrorIs(t, run([]string{"-h"}, &bytes.Buffer{}), flag.ErrHelp)
}

func TestInvalidPNG(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "invalid.png"), filepath.Join(dir, "output.unity3d")
	require.NoError(t, os.WriteFile(input, []byte("invalid"), 0o600))
	require.NoError(t, os.WriteFile(output, []byte("existing"), 0o600))
	args := []string{"-i", input, "-o", output, "-p", "ios", "-r", "700101012"}
	require.ErrorContains(t, run(args, &bytes.Buffer{}), "read PNG")
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, img))
	data := encoded.Bytes()
	binary.BigEndian.PutUint32(data[16:], 20000)
	binary.BigEndian.PutUint32(data[29:], crc32.ChecksumIEEE(data[12:29]))
	require.NoError(t, os.WriteFile(input, data, 0o600))
	require.ErrorContains(t, run(args, &bytes.Buffer{}), "size limit")
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	require.Equal(t, "existing", string(data))
}
