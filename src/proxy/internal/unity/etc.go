// Translated from K0lb3/texture2ddecoder's ETC decoder.

package unity

import (
	"encoding/binary"
	"fmt"
	"image"
)

var etcModifiers = [8][2]int{{2, 8}, {5, 17}, {9, 29}, {13, 42}, {18, 60}, {24, 80}, {33, 106}, {47, 183}}
var etcDistances = [8]int{3, 6, 11, 16, 23, 32, 41, 64}
var etcAlphaModifiers = [16][8]int{
	{-3, -6, -9, -15, 2, 5, 8, 14}, {-3, -7, -10, -13, 2, 6, 9, 12}, {-2, -5, -8, -13, 1, 4, 7, 12},
	{-2, -4, -6, -13, 1, 3, 5, 12}, {-3, -6, -8, -12, 2, 5, 7, 11}, {-3, -7, -9, -11, 2, 6, 8, 10},
	{-4, -7, -8, -11, 3, 6, 7, 10}, {-3, -5, -8, -11, 2, 4, 7, 10}, {-2, -6, -8, -10, 1, 5, 7, 9},
	{-2, -5, -8, -10, 1, 4, 7, 9}, {-2, -4, -8, -10, 1, 3, 7, 9}, {-2, -5, -7, -10, 1, 4, 6, 9},
	{-3, -4, -7, -10, 2, 3, 6, 9}, {-1, -2, -3, -10, 0, 1, 2, 9}, {-4, -6, -8, -9, 3, 5, 7, 8},
	{-3, -5, -7, -9, 2, 4, 6, 8},
}

func etcColor(c [3]int, modifier int) [4]byte {
	return [4]byte{byte(astcClamp(c[0] + modifier)), byte(astcClamp(c[1] + modifier)), byte(astcClamp(c[2] + modifier)), 255}
}

func decodeETCBlock(data []byte, extended bool) [16][4]byte {
	var out [16][4]byte
	d := [8]int{}
	for i := range d {
		d[i] = int(data[i])
	}
	var c [3][3]int
	differential := d[3]&2 != 0
	if differential {
		var delta [3]int
		for ch := range 3 {
			c[0][ch] = d[ch] & 0xf8
			delta[ch] = (d[ch] << 3 & 0x18) - (d[ch] << 3 & 0x20)
			c[1][ch] = c[0][ch] + delta[ch]
		}
		var palette [4][4]byte
		paletteMode := false
		if extended && (c[1][0] < 0 || c[1][0] > 255) {
			c[0] = [3]int{(d[0] << 3 & 0xc0) | (d[0] << 4 & 0x30) | (d[0] >> 1 & 0xc) | (d[0] & 3), (d[1] & 0xf0) | d[1]>>4, (d[1] & 15) * 17}
			c[1] = [3]int{(d[2] & 0xf0) | d[2]>>4, (d[2] & 15) * 17, (d[3] & 0xf0) | d[3]>>4}
			distance := etcDistances[(d[3]>>1&6)|(d[3]&1)]
			palette = [4][4]byte{etcColor(c[0], 0), etcColor(c[1], distance), etcColor(c[1], 0), etcColor(c[1], -distance)}
			paletteMode = true
		} else if extended && (c[1][1] < 0 || c[1][1] > 255) {
			c[0][0] = (d[0] << 1 & 0xf0) | (d[0] >> 3 & 15)
			c[0][1] = (d[0] << 5 & 0xe0) | (d[1] & 0x10)
			c[0][1] |= c[0][1] >> 4
			c[0][2] = (d[1] & 8) | (d[1] << 1 & 6) | d[2]>>7
			c[0][2] *= 17
			c[1][0] = (d[2] << 1 & 0xf0) | (d[2] >> 3 & 15)
			c[1][1] = (d[2] << 5 & 0xe0) | (d[3] >> 3 & 0x10)
			c[1][1] |= c[1][1] >> 4
			c[1][2] = (d[3] << 1 & 0xf0) | (d[3] >> 3 & 15)
			i := (d[3] & 4) | (d[3] << 1 & 2)
			if c[0][0] > c[1][0] || (c[0][0] == c[1][0] && (c[0][1] > c[1][1] || (c[0][1] == c[1][1] && c[0][2] >= c[1][2]))) {
				i++
			}
			distance := etcDistances[i]
			palette = [4][4]byte{etcColor(c[0], distance), etcColor(c[0], -distance), etcColor(c[1], distance), etcColor(c[1], -distance)}
			paletteMode = true
		} else if extended && (c[1][2] < 0 || c[1][2] > 255) {
			c[0][0] = (d[0] << 1 & 0xfc) | (d[0] >> 5 & 3)
			c[0][1] = (d[0] << 7 & 0x80) | (d[1] & 0x7e) | (d[0] & 1)
			c[0][2] = (d[1] << 7 & 0x80) | (d[2] << 2 & 0x60) | (d[2] << 3 & 0x18) | (d[3] >> 5 & 4)
			c[0][2] |= c[0][2] >> 6
			c[1][0] = (d[3] << 1 & 0xf8) | (d[3] << 2 & 4) | (d[3] >> 5 & 3)
			c[1][1] = (d[4] & 0xfe) | d[4]>>7
			c[1][2] = (d[4] << 7 & 0x80) | (d[5] >> 1 & 0x7c)
			c[1][2] |= c[1][2] >> 6
			c[2][0] = (d[5] << 5 & 0xe0) | (d[6] >> 3 & 0x1c) | (d[5] >> 1 & 3)
			c[2][1] = (d[6] << 3 & 0xf8) | (d[7] >> 5 & 6) | (d[6] >> 4 & 1)
			c[2][2] = (d[7] << 2 & 255) | (d[7] >> 4 & 3)
			for y := range 4 {
				for x := range 4 {
					for ch := range 3 {
						out[y*4+x][ch] = byte(astcClamp((x*(c[1][ch]-c[0][ch]) + y*(c[2][ch]-c[0][ch]) + 4*c[0][ch] + 2) >> 2))
					}
					out[y*4+x][3] = 255
				}
			}
			return out
		}
		if paletteMode {
			j, k := d[6]<<8|d[7], d[4]<<8|d[5]
			for i := range 16 {
				out[i%4*4+i/4] = palette[((k>>i&1)<<1)|(j>>i&1)]
			}
			return out
		}
		for s := range 2 {
			for ch := range 3 {
				v := c[s][ch] & 255
				c[s][ch] = v | v>>5
			}
		}
	} else {
		for ch := range 3 {
			c[0][ch], c[1][ch] = (d[ch]>>4)*17, (d[ch]&15)*17
		}
	}
	code := [2]int{d[3] >> 5, d[3] >> 2 & 7}
	j, k := d[6]<<8|d[7], d[4]<<8|d[5]
	for i := range 16 {
		s := i / 8
		if d[3]&1 != 0 {
			s = i % 4 / 2
		}
		modifier := etcModifiers[code[s]][j>>i&1]
		if k>>i&1 != 0 {
			modifier = -modifier
		}
		out[i%4*4+i/4] = etcColor(c[s], modifier)
	}
	return out
}

func decodeCompressed(t *texture) (image.Image, error) {
	blockSize := 8
	switch t.format {
	case 34, 45:
	case 47:
		blockSize = 16
	default:
		return nil, fmt.Errorf("unsupported Unity texture format %d", t.format)
	}
	xBlocks, yBlocks := (t.width+3)/4, (t.height+3)/4
	if len(t.data) < xBlocks*yBlocks*blockSize {
		return nil, fmt.Errorf("truncated ETC texture")
	}
	img := image.NewNRGBA(image.Rect(0, 0, t.width, t.height))
	for by := range yBlocks {
		for bx := range xBlocks {
			data := t.data[(by*xBlocks+bx)*blockSize:][:blockSize]
			pixels := decodeETCBlock(data[blockSize-8:], t.format != 34)
			if t.format == 47 {
				indices := binary.BigEndian.Uint64(data)
				for i := range 16 {
					alpha := int(data[0]) + int(data[1]>>4)*etcAlphaModifiers[data[1]&15][indices>>uint(i*3)&7]
					pixels[15-(i%4*4+i/4)][3] = byte(astcClamp(alpha))
				}
			}
			for y := 0; y < 4 && by*4+y < t.height; y++ {
				for x := 0; x < 4 && bx*4+x < t.width; x++ {
					dst := img.Pix[((t.height-1-by*4-y)*t.width+bx*4+x)*4:]
					copy(dst, pixels[y*4+x][:])
				}
			}
		}
	}
	return img, nil
}
