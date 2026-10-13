package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	reference, err := os.ReadFile("testdata/reference.unity3d")
	require.NoError(t, err)
	for _, platform := range []string{"ios", "android"} {
		t.Run(platform, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "output.unity3d")
			require.NoError(t, run([]string{"-i", "testdata/input.png", "-o", output, "-p", platform, "-r", "700101012"}, &bytes.Buffer{}))
			actual, err := os.ReadFile(output)
			require.NoError(t, err)
			expected := bytes.Clone(reference)
			if platform == "android" {
				asset := bytes.LastIndex(expected, []byte(unityVersion+"\x00"))
				require.Greater(t, asset, 0)
				binary.LittleEndian.PutUint32(expected[asset+len(unityVersion)+1:], 13)
			}
			require.Equal(t, expected, actual)
		})
	}
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
