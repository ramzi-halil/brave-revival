package unity

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/pierrec/lz4/v4"
	"github.com/stretchr/testify/require"
)

func TestPixelVectors(t *testing.T) {
	data, err := os.ReadFile("testdata/pixels.json")
	require.NoError(t, err)
	var vectors []struct {
		Name, Data, SHA256    string
		Width, Height, Format int
	}
	require.NoError(t, json.Unmarshal(data, &vectors))
	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			data, err := hex.DecodeString(v.Data)
			require.NoError(t, err)
			tex := &texture{v.Width, v.Height, v.Format, data}
			img, err := decodeTexture(tex)
			require.NoError(t, err)
			digest := sha256.Sum256(img.(*image.NRGBA).Pix)
			require.Equal(t, v.SHA256, hex.EncodeToString(digest[:]))
			tex.data = data[:len(data)-1]
			_, err = decodeTexture(tex)
			require.Error(t, err)
		})
	}
}

func TestASTCAliases(t *testing.T) {
	block := []byte{0xfc, 0xfd, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0xff, 0xff}
	for format := 48; format <= 59; format++ {
		img, err := decodeTexture(&texture{5, 3, format, bytes.Repeat(block, 2)})
		require.NoError(t, err)
		require.Equal(t, color.NRGBA{R: 255, A: 255}, color.NRGBAModel.Convert(img.At(0, 0)))
		require.Equal(t, color.NRGBA{R: 255, A: 255}, color.NRGBAModel.Convert(img.At(4, 2)))
	}
	_, err := decodeTexture(&texture{5, 5, 49, make([]byte, 16)})
	require.Error(t, err)
}

func TestRawPixels(t *testing.T) {
	for _, tc := range []struct {
		format int
		data   []byte
		want   color.NRGBA
	}{
		{1, []byte{128}, color.NRGBA{255, 255, 255, 128}},
		{3, []byte{10, 20, 30}, color.NRGBA{10, 20, 30, 255}},
		{4, []byte{10, 20, 30, 128}, color.NRGBA{10, 20, 30, 128}},
		{5, []byte{128, 10, 20, 30}, color.NRGBA{10, 20, 30, 128}},
		{14, []byte{30, 20, 10, 128}, color.NRGBA{10, 20, 30, 128}},
	} {
		img, err := decodeTexture(&texture{1, 1, tc.format, tc.data})
		require.NoError(t, err)
		require.Equal(t, tc.want, color.NRGBAModel.Convert(img.At(0, 0)))
	}
	for _, tex := range []*texture{{-1, 1, 4, nil}, {16384, 16384, 4, nil}, {1, 1, 4, nil}, {1, 1, 999, nil}} {
		_, err := decodeTexture(tex)
		require.Error(t, err)
	}
}

func TestBundles(t *testing.T) {
	data, err := os.ReadFile("testdata/texture.unity3d")
	require.NoError(t, err)
	files, err := readBundle(data)
	require.NoError(t, err)
	_, err = TexturePNG(files[0].data, files[0].name)
	require.ErrorContains(t, err, "streamed texture data not found")
	for _, flags := range []uint32{0, 2, 3, 0x82, 0x202, 0x282} {
		bundle := testBundle(t, files, flags)
		encoded, err := TexturePNG(bundle, "test")
		require.NoError(t, err)
		img, err := png.Decode(bytes.NewReader(encoded))
		require.NoError(t, err)
		require.Equal(t, color.NRGBA{R: 255, A: 255}, color.NRGBAModel.Convert(img.At(0, 0)))
	}
	files[1].data = files[1].data[:1]
	_, err = TexturePNG(testBundle(t, files, 2), "test")
	require.ErrorContains(t, err, "outside")
}

func TestTypeTrees(t *testing.T) {
	for _, name := range []string{"tree-le.assets", "tree-be.assets"} {
		data, err := os.ReadFile("testdata/" + name)
		require.NoError(t, err)
		encoded, err := TexturePNG(data, name)
		require.NoError(t, err)
		img, err := png.Decode(bytes.NewReader(encoded))
		require.NoError(t, err)
		require.Equal(t, color.NRGBA{R: 255, A: 255}, color.NRGBAModel.Convert(img.At(0, 0)))
		require.Equal(t, color.NRGBA{R: 255, G: 255, B: 255, A: 128}, color.NRGBAModel.Convert(img.At(1, 1)))
		empty, err := os.ReadFile("testdata/no-texture.assets")
		require.NoError(t, err)
		_, err = TexturePNG(testBundle(t, []bundleFile{{"empty", empty}, {name, data}}, 2), "test")
		require.NoError(t, err)
	}
}

func testBundle(t *testing.T, files []bundleFile, flags uint32) []byte {
	t.Helper()
	var payload, info, header bytes.Buffer
	write := func(b *bytes.Buffer, values ...any) {
		for _, v := range values {
			require.NoError(t, binary.Write(b, binary.BigEndian, v))
		}
	}
	compress := func(data []byte) []byte {
		if flags&63 == 0 {
			return data
		}
		out := make([]byte, lz4.CompressBlockBound(len(data)))
		n, err := lz4.CompressBlock(data, out, nil)
		require.NoError(t, err)
		if n == 0 {
			out = []byte{0xf0}
			for remaining := len(data) - 15; ; remaining -= 255 {
				out = append(out, byte(min(remaining, 255)))
				if remaining < 255 {
					break
				}
			}
			return append(out, data...)
		}
		return out[:n]
	}
	for _, f := range files {
		payload.Write(f.data)
	}
	block := compress(payload.Bytes())
	write(&info, [16]byte{}, uint32(1), uint32(payload.Len()), uint32(len(block)), uint16(flags&63), uint32(len(files)))
	offset := 0
	for _, f := range files {
		write(&info, uint64(offset), uint64(len(f.data)), uint32(0))
		info.WriteString(f.name + "\x00")
		offset += len(f.data)
	}
	compressedInfo := compress(info.Bytes())
	header.WriteString("UnityFS\x00")
	write(&header, uint32(8))
	header.WriteString("5.x.x\x002022.3.62f3\x00")
	sizeOffset := header.Len()
	write(&header, uint64(0), uint32(len(compressedInfo)), uint32(info.Len()), flags|0x40)
	for header.Len()%16 != 0 {
		header.WriteByte(0)
	}
	if flags&0x80 == 0 {
		header.Write(compressedInfo)
	}
	if flags&0x200 != 0 {
		for header.Len()%16 != 0 {
			header.WriteByte(0)
		}
	}
	header.Write(block)
	if flags&0x80 != 0 {
		header.Write(compressedInfo)
	}
	data := header.Bytes()
	binary.BigEndian.PutUint64(data[sizeOffset:], uint64(len(data)))
	return data
}

func FuzzTexturePNG(f *testing.F) {
	for _, name := range []string{"testdata/texture.unity3d", "testdata/texture.assets", "testdata/no-texture.assets", "testdata/tree-le.assets", "testdata/tree-be.assets"} {
		data, err := os.ReadFile(name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) { TexturePNG(data, "test") })
}
