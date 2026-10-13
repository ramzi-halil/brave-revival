// Serialization layouts adapted from UnityPy (MIT); see ../../NOTICE.txt.

package main

import (
	"encoding/binary"
	"image"
	"image/draw"
	"strings"
)

const (
	unityVersion = "2022.3.62f3"
	rgba32       = 4
)

type writer struct {
	data  []byte
	order binary.AppendByteOrder
}

func (w *writer) u8(v byte)        { w.data = append(w.data, v) }
func (w *writer) u16(v uint16)     { w.data = w.order.AppendUint16(w.data, v) }
func (w *writer) u32(v uint32)     { w.data = w.order.AppendUint32(w.data, v) }
func (w *writer) u64(v uint64)     { w.data = w.order.AppendUint64(w.data, v) }
func (w *writer) raw(v []byte)     { w.data = append(w.data, v...) }
func (w *writer) cstring(v string) { w.raw([]byte(v)); w.u8(0) }
func (w *writer) align(n int) {
	for len(w.data)%n != 0 {
		w.u8(0)
	}
}
func (w *writer) string(v string)   { w.u32(uint32(len(v))); w.raw([]byte(v)); w.align(4) }
func (w *writer) pointer(id uint64) { w.u32(0); w.u64(id) }

func textureObject(img image.Image, name string) []byte {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	pixels := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(pixels, pixels.Bounds(), img, bounds.Min, draw.Src)
	w := writer{order: binary.LittleEndian}
	w.string(name)
	w.u32(rgba32)
	w.u8(0)
	w.u8(0)
	w.align(4)
	w.u32(uint32(width))
	w.u32(uint32(height))
	w.u32(uint32(len(pixels.Pix)))
	w.u32(0)
	w.u32(rgba32)
	w.u32(1)
	w.u8(0)
	w.u8(0)
	w.u8(0)
	w.align(4)
	w.string("")
	w.u8(0)
	w.align(4)
	w.u32(0)
	w.u32(1)
	w.u32(2)
	w.u32(1)
	w.u32(1)
	w.u32(0)
	w.u32(1)
	w.u32(1)
	w.u32(1)
	w.u32(0)
	w.u32(1)
	w.u32(0)
	w.u32(uint32(len(pixels.Pix)))
	// Unity's texture rows run from bottom to top; PNG rows run from top to bottom.
	for y := height - 1; y >= 0; y-- {
		w.raw(pixels.Pix[y*pixels.Stride : y*pixels.Stride+width*4])
	}
	w.align(4)
	w.u64(0)
	w.u32(0)
	w.string("")
	return w.data
}

func assetBundleObject(name string) []byte {
	w := writer{order: binary.LittleEndian}
	w.string(name)
	w.u32(1)
	w.pointer(2)
	w.u32(1)
	w.string(strings.ToLower(name) + ".png")
	w.u32(0)
	w.u32(1)
	w.pointer(2)
	w.u32(0)
	w.u32(0)
	w.pointer(0)
	w.u32(1)
	w.string(name)
	w.u32(0)
	w.u8(0)
	w.align(4)
	w.u32(0)
	w.u32(7)
	w.u32(0)
	return w.data
}

func serializedAsset(img image.Image, name string, platform int32) []byte {
	objects := [][]byte{assetBundleObject(name), textureObject(img, name)}
	metadata := writer{order: binary.LittleEndian}
	metadata.cstring(unityVersion)
	metadata.u32(uint32(platform))
	metadata.u8(0)
	metadata.u32(2)
	// Built-in type hashes for the game's Unity 2022.3 layouts, without type trees.
	for _, typ := range []struct {
		class uint32
		hash  [16]byte
	}{
		{142, [16]byte{0x97, 0xda, 0x5f, 0x46, 0x88, 0xe4, 0x5a, 0x57, 0xc8, 0xb4, 0x2d, 0x4f, 0x42, 0x49, 0x72, 0x97}},
		{28, [16]byte{0x31, 0xf6, 0x5a, 0xe3, 0x43, 0xf6, 0x71, 0xbf, 0x46, 0x2f, 0x15, 0xc1, 0xb4, 0x1e, 0x97, 0x68}},
	} {
		metadata.u32(typ.class)
		metadata.u8(0)
		metadata.u16(0xffff)
		metadata.raw(typ.hash[:])
	}
	metadata.u32(uint32(len(objects)))
	data := writer{order: binary.LittleEndian}
	for index, object := range objects {
		metadata.align(4)
		metadata.u64(uint64(index + 1))
		metadata.u64(uint64(len(data.data)))
		metadata.u32(uint32(len(object)))
		metadata.u32(uint32(index))
		data.raw(object)
		data.align(8)
	}
	metadata.u32(0)
	metadata.u32(0)
	metadata.u32(0)
	metadata.cstring("")
	offset := (48 + len(metadata.data) + 15) &^ 15
	w := writer{order: binary.BigEndian}
	w.u32(0)
	w.u32(0)
	w.u32(22)
	w.u32(0)
	w.u32(0)
	w.u32(uint32(len(metadata.data)))
	w.u64(uint64(offset + len(data.data)))
	w.u64(uint64(offset))
	w.u64(0)
	w.raw(metadata.data)
	w.align(16)
	w.raw(data.data)
	return w.data
}

func textureBundle(img image.Image, name string, platform int32) []byte {
	asset := serializedAsset(img, name, platform)
	info := writer{order: binary.BigEndian}
	info.raw(make([]byte, 16))
	info.u32(1)
	info.u32(uint32(len(asset)))
	info.u32(uint32(len(asset)))
	info.u16(0)
	info.u32(1)
	info.u64(0)
	info.u64(uint64(len(asset)))
	info.u32(4)
	info.cstring("CAB-" + name + "r")
	w := writer{order: binary.BigEndian}
	w.cstring("UnityFS")
	w.u32(8)
	w.cstring("5.x.x")
	w.cstring(unityVersion)
	sizeOffset := len(w.data)
	w.u64(0)
	w.u32(uint32(len(info.data)))
	w.u32(uint32(len(info.data)))
	w.u32(0x40)
	w.align(16)
	w.raw(info.data)
	w.raw(asset)
	binary.BigEndian.PutUint64(w.data[sizeOffset:], uint64(len(w.data)))
	return w.data
}
