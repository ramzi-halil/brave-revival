// Adapted from UnityPy's Texture2D type trees and Texture2DConverter.py (MIT); see LICENSE.UnityPy.

package unity

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"path"
)

type texture struct {
	width, height, format int
	data                  []byte
}

func TexturePNG(data []byte, name string) (result []byte, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			result = nil
			err = fmt.Errorf("failed to decode Unity texture: %v", failure)
		}
	}()
	files := []bundleFile{{name, data}}
	if bytes.HasPrefix(data, []byte("UnityFS\x00")) || bytes.HasPrefix(data, []byte("UnityRaw\x00")) || bytes.HasPrefix(data, []byte("UnityWeb\x00")) {
		files, err = readBundle(data)
		if err != nil {
			return nil, err
		}
	} else if !isSerialized(data) {
		return nil, fmt.Errorf("unsupported Unity asset file")
	}
	streams := make(map[string][]byte)
	for _, file := range files {
		streams[path.Base(file.name)] = file.data
	}
	for _, file := range files {
		if !isSerialized(file.data) {
			continue
		}
		texture, err := readAsset(file.data, streams)
		if err != nil {
			return nil, err
		}
		if texture == nil {
			continue
		}
		img, err := decodeTexture(texture)
		if err != nil {
			return nil, err
		}
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, img); err != nil {
			return nil, err
		}
		return encoded.Bytes(), nil
	}
	return nil, fmt.Errorf("asset contains no Texture2D")
}

func textureFromValues(values map[string]any, streams map[string][]byte) *texture {
	number := func(name string) int { return int(values[name].(int64)) }
	t := &texture{width: number("m_Width"), height: number("m_Height"), format: number("m_TextureFormat")}
	t.data, _ = values["image data"].([]byte)
	if len(t.data) == 0 {
		if info, ok := values["m_StreamData"].(map[string]any); ok {
			t.data = streamData(info["path"].(string), uint64(info["offset"].(int64)), uint64(info["size"].(int64)), streams)
		}
	}
	return t
}

func streamData(name string, offset, size uint64, streams map[string][]byte) []byte {
	data, ok := streams[path.Base(name)]
	if !ok {
		panic(fmt.Errorf("streamed texture data not found"))
	}
	return slice(data, offset, size)
}

func readTexture(r *reader, v unityVersion, platform int, streams map[string][]byte) *texture {
	if !v.atLeast(5, 3, 0) || platform == -2 {
		panic(fmt.Errorf("unsupported Unity Texture2D layout without type tree"))
	}
	r.string()
	if v.atLeast(2017, 3, 0) {
		r.u32()
		r.u8()
		if v.atLeast(2020, 2, 0) {
			r.u8()
		}
		r.align(4)
	}
	t := &texture{width: r.i32(), height: r.i32()}
	r.u32()
	if v.atLeast(2020, 1, 0) {
		r.u32()
	}
	t.format = r.i32()
	r.u32()
	r.u8()
	if v.atLeast(2020, 1, 0) {
		r.u8()
	}
	if v.atLeast(2019, 3, 0) {
		r.u8()
	}
	if v.atLeast(2022, 2, 0) {
		r.align(4)
		r.string()
	}
	if !v.atLeast(5, 5, 0) || v.atLeast(2018, 2, 0) {
		r.u8()
	}
	r.align(4)
	if v.atLeast(2018, 2, 0) {
		r.u32()
	}
	r.take(8)
	r.take(16)
	if v.atLeast(2017, 0, 0) {
		r.take(8)
	}
	r.take(8)
	if v.atLeast(2020, 2, 0) {
		r.byteArray()
		r.align(4)
	}
	t.data = r.byteArray()
	if len(t.data) == 0 {
		var offset uint64
		if v.atLeast(2020, 1, 0) {
			offset = r.u64()
		} else {
			offset = uint64(r.u32())
		}
		size := uint64(r.u32())
		t.data = streamData(r.string(), offset, size, streams)
	}
	return t
}

func decodeTexture(t *texture) (image.Image, error) {
	if t.width <= 0 || t.height <= 0 || t.width > 16384 || t.height > 16384 || t.width*t.height > maxAssetSize/4 {
		return nil, fmt.Errorf("invalid texture dimensions")
	}
	if t.format >= 48 && t.format <= 59 {
		block := [...]int{4, 5, 6, 8, 10, 12}[(t.format-48)%6]
		return decodeASTC(t.data, t.width, t.height, block, block)
	}
	channels := 4
	switch t.format {
	case 1:
		channels = 1
	case 3:
		channels = 3
	case 4, 5, 14:
	default:
		return decodeCompressed(t)
	}
	if len(t.data) < t.width*t.height*channels {
		return nil, fmt.Errorf("truncated texture pixels")
	}
	img := image.NewNRGBA(image.Rect(0, 0, t.width, t.height))
	for y := range t.height {
		for x := range t.width {
			src := t.data[(y*t.width+x)*channels:]
			dst := img.Pix[((t.height-1-y)*t.width+x)*4:]
			switch t.format {
			case 1:
				copy(dst, []byte{255, 255, 255, src[0]})
			case 3:
				copy(dst, src[:3])
				dst[3] = 255
			case 4:
				copy(dst, src[:4])
			case 5:
				copy(dst, []byte{src[1], src[2], src[3], src[0]})
			case 14:
				copy(dst, []byte{src[2], src[1], src[0], src[3]})
			}
		}
	}
	return img, nil
}
