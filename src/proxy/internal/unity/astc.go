// Translated from K0lb3/texture2ddecoder's MIT-licensed ASTC decoder.
// See LICENSE.texture2ddecoder and README.md for upstream attribution.

package unity

import (
	"encoding/binary"
	"fmt"
	"image"
	"math/bits"
)

var weightA = [...]int{0, 0, 0, 3, 0, 5, 3, 0, 0, 0, 5, 3, 0, 5, 3, 0}
var weightB = [...]int{0, 0, 1, 0, 2, 0, 1, 3, 0, 0, 1, 2, 4, 2, 3, 5}
var endpointA = [...]int{0, 3, 5, 0, 3, 5, 0, 3, 5, 0, 3, 5, 0, 3, 5, 0, 3, 0, 0}
var endpointB = [...]int{8, 6, 5, 7, 5, 4, 6, 4, 3, 5, 3, 2, 4, 2, 1, 3, 1, 2, 1}

type astcBlock struct {
	bw, bh, width, height, parts, planes, selector, weightRange, weightCount, endpointRange, endpointCount int
	mode                                                                                                   [4]int
	endpoints                                                                                              [4][8]int
	weights                                                                                                [144][2]int
	partition                                                                                              [144]int
}

func astcBits(data []byte, pos, length int) uint64 {
	var value uint64
	for i := 0; i < length; {
		bit := pos + i
		if bit < 0 {
			i++
			continue
		}
		if bit >= 128 {
			break
		}
		n := min(length-i, 8-bit%8)
		value |= uint64((int(data[bit/8])>>(bit%8))&((1<<n)-1)) << i
		i += n
	}
	return value
}

type astcInt struct{ bits, nonbits int }

func astcSequence(data []byte, offset, a, b, count int, reverse bool) []astcInt {
	out := make([]astcInt, count)
	if a == 0 {
		for i := range count {
			pos := offset + i*b
			if reverse {
				pos = offset - (i+1)*b
			}
			v := astcBits(data, pos, b)
			if reverse {
				v = bits.Reverse64(v) >> (64 - b)
			}
			out[i].bits = int(v)
		}
		return out
	}
	group, extra := 5, 8
	positions := []int{0, 2, 4, 5, 7}
	if a == 5 {
		group, extra = 3, 7
		positions = []int{0, 3, 5}
	}
	blockSize := extra + group*b
	for i, n := 0, 0; n < count; i++ {
		now := min(group, count-n)
		size := (blockSize*now + group - 1) / group
		pos := offset + i*blockSize
		if reverse {
			pos = offset - i*blockSize - size
		}
		value := astcBits(data, pos, size)
		if reverse {
			value = bits.Reverse64(value) >> (64 - size)
		}
		d := int(value)
		x := (d >> b & 3) | (d >> (b * 2) & 0xc) | (d >> (b * 3) & 0x10) | (d >> (b * 4) & 0x60) | (d >> (b * 5) & 0x80)
		if a == 5 {
			x = (d >> b & 7) | (d >> (b * 2) & 0x18) | (d >> (b * 3) & 0x60)
		}
		for j := range now {
			v := int(value>>(positions[j]+b*j)) & ((1 << b) - 1)
			if a == 3 {
				out[n] = astcInt{v, astcTrits[j][x]}
			} else {
				out[n] = astcInt{v, astcQuints[j][x]}
			}
			n++
		}
	}
	return out
}

func sequenceBits(count, a, b int) int {
	size := count * b
	if a == 3 {
		size += (count*8 + 4) / 5
	} else if a == 5 {
		size += (count*7 + 2) / 3
	}
	return size
}

func (d *astcBlock) parameters(buf []byte) error {
	u0, u1 := int(buf[0]), int(buf[1])
	u16 := int(binary.LittleEndian.Uint16(buf))
	d.planes = 1
	if u1&4 != 0 {
		d.planes = 2
	}
	d.weightRange = (u0 >> 4 & 1) | (u1 << 2 & 8)
	if u0&3 != 0 {
		d.weightRange |= u0 << 1 & 6
		switch u0 & 12 {
		case 0:
			d.width, d.height = (u16>>7&3)+4, (u0>>5&3)+2
		case 4:
			d.width, d.height = (u16>>7&3)+8, (u0>>5&3)+2
		case 8:
			d.width, d.height = (u0>>5&3)+2, (u16>>7&3)+8
		case 12:
			if u1&1 != 0 {
				d.width, d.height = (u0>>7&1)+2, (u0>>5&3)+2
			} else {
				d.width, d.height = (u0>>5&3)+2, (u0>>7&1)+6
			}
		}
	} else {
		d.weightRange |= u0 >> 1 & 6
		switch u16 & 0x180 {
		case 0:
			d.width, d.height = 12, (u0>>5&3)+2
		case 0x80:
			d.width, d.height = (u0>>5&3)+2, 12
		case 0x100:
			d.width, d.height = (u0>>5&3)+6, (u1>>1&3)+6
			d.planes = 1
			d.weightRange &= 7
		case 0x180:
			d.width, d.height = 6, 10
			if u0&0x20 != 0 {
				d.width, d.height = 10, 6
			}
		}
	}
	d.parts = (u1 >> 3 & 3) + 1
	d.weightCount = d.width * d.height * d.planes
	if d.weightCount > 64 || d.width > d.bw || d.height > d.bh || (d.planes == 2 && d.parts == 4) {
		return fmt.Errorf("invalid ASTC weight grid")
	}
	weightBits := sequenceBits(d.weightCount, weightA[d.weightRange], weightB[d.weightRange])
	configBits, base := 17, 0
	if d.parts == 1 {
		d.mode[0] = int(binary.LittleEndian.Uint16(buf[1:])) >> 5 & 15
	} else {
		base = int(binary.LittleEndian.Uint16(buf[2:])) >> 7 & 3
		if base == 0 {
			for i := range d.parts {
				d.mode[i] = int(buf[3]) >> 1 & 15
			}
			configBits = 29
		} else {
			for i := range d.parts {
				d.mode[i] = ((int(buf[3]) >> (i + 1) & 1) + base - 1) << 2
			}
			switch d.parts {
			case 2:
				d.mode[0] |= int(buf[3]) >> 3 & 3
				d.mode[1] |= int(astcBits(buf, 126-weightBits, 2))
			case 3:
				d.mode[0] |= int(buf[3]) >> 4 & 1
				d.mode[0] |= int(astcBits(buf, 122-weightBits, 2)) & 2
				d.mode[1] |= int(astcBits(buf, 124-weightBits, 2))
				d.mode[2] |= int(astcBits(buf, 126-weightBits, 2))
			case 4:
				for i := range 4 {
					d.mode[i] |= int(astcBits(buf, 120+i*2-weightBits, 2))
				}
			}
			configBits = 25 + d.parts*3
		}
	}
	if d.planes == 2 {
		configBits += 2
		pos := 126 - weightBits
		if base != 0 {
			pos = 130 - weightBits - d.parts*3
		}
		d.selector = int(astcBits(buf, pos, 2))
	}
	for i := range d.parts {
		d.endpointCount += (d.mode[i] >> 1 & 6) + 2
	}
	d.endpointRange = -1
	for i := range endpointA {
		if sequenceBits(d.endpointCount, endpointA[i], endpointB[i]) <= 128-configBits-weightBits {
			d.endpointRange = i
			break
		}
	}
	if d.endpointRange < 0 || weightBits < 24 || weightBits > 96 {
		return fmt.Errorf("invalid ASTC bit allocation")
	}
	return nil
}

func astcClamp(v int) int { return min(255, max(0, v)) }

func transferSigned(a, b *int) {
	*b = (*b >> 1) | (*a & 0x80)
	*a = (*a >> 1) & 0x3f
	if *a&0x20 != 0 {
		*a -= 0x40
	}
}

func (d *astcBlock) decodeEndpoints(buf []byte) error {
	a, b := endpointA[d.endpointRange], endpointB[d.endpointRange]
	offset := 17
	if d.parts > 1 {
		offset = 29
	}
	seq := astcSequence(buf, offset, a, b, d.endpointCount, false)
	var ev [32]int
	for i, s := range seq {
		if a != 0 {
			mask, x, replicated, c := (s.bits&1)*0x1ff, s.bits>>1, 0, 0
			if a == 3 {
				c = [...]int{0, 204, 93, 44, 22, 11, 5}[b]
				switch b {
				case 2:
					replicated = 0b100010110 * x
				case 3:
					replicated = x<<7 | x<<2 | x
				case 4:
					replicated = x<<6 | x
				case 5:
					replicated = x<<5 | x>>2
				case 6:
					replicated = x<<4 | x>>4
				}
			} else {
				c = [...]int{0, 113, 54, 26, 13, 6}[b]
				switch b {
				case 2:
					replicated = 0b100001100 * x
				case 3:
					replicated = x<<7 | x<<1 | x>>1
				case 4:
					replicated = x<<6 | x>>1
				case 5:
					replicated = x<<5 | x>>3
				}
			}
			ev[i] = (mask & 0x80) | (((s.nonbits*c + replicated) ^ mask) >> 2)
		} else {
			x := s.bits
			switch b {
			case 1:
				ev[i] = x * 255
			case 2:
				ev[i] = x * 85
			case 3:
				ev[i] = x<<5 | x<<2 | x>>1
			case 4:
				ev[i] = x<<4 | x
			case 5:
				ev[i] = x<<3 | x>>2
			case 6:
				ev[i] = x<<2 | x>>4
			case 7:
				ev[i] = x<<1 | x>>6
			case 8:
				ev[i] = x
			}
		}
	}
	pos := 0
	for part := range d.parts {
		mode := d.mode[part]
		v := ev[pos:]
		pos += (mode/4 + 1) * 2
		var e [8]int
		blue, clamp := false, false
		switch mode {
		case 0:
			e = [8]int{v[0], v[0], v[0], 255, v[1], v[1], v[1], 255}
		case 1:
			lo := (v[0] >> 2) | (v[1] & 0xc0)
			hi := astcClamp(lo + (v[1] & 0x3f))
			e = [8]int{lo, lo, lo, 255, hi, hi, hi, 255}
		case 4:
			e = [8]int{v[0], v[0], v[0], v[2], v[1], v[1], v[1], v[3]}
		case 5:
			transferSigned(&v[1], &v[0])
			transferSigned(&v[3], &v[2])
			hi := v[0] + v[1]
			e = [8]int{v[0], v[0], v[0], v[2], hi, hi, hi, v[2] + v[3]}
			clamp = true
		case 6:
			e = [8]int{(v[0] * v[3]) >> 8, (v[1] * v[3]) >> 8, (v[2] * v[3]) >> 8, 255, v[0], v[1], v[2], 255}
		case 8, 12:
			e = [8]int{v[0], v[2], v[4], 255, v[1], v[3], v[5], 255}
			if mode == 12 {
				e[3], e[7] = v[6], v[7]
			}
			blue = v[0]+v[2]+v[4] > v[1]+v[3]+v[5]
			if blue {
				for i := range 4 {
					e[i], e[i+4] = e[i+4], e[i]
				}
			}
		case 9, 13:
			transferSigned(&v[1], &v[0])
			transferSigned(&v[3], &v[2])
			transferSigned(&v[5], &v[4])
			e = [8]int{v[0], v[2], v[4], 255, v[0] + v[1], v[2] + v[3], v[4] + v[5], 255}
			if mode == 13 {
				transferSigned(&v[7], &v[6])
				e[3], e[7] = v[6], v[6]+v[7]
			}
			blue, clamp = v[1]+v[3]+v[5] < 0, true
			if blue {
				for i := range 4 {
					e[i], e[i+4] = e[i+4], e[i]
				}
			}
		case 10:
			e = [8]int{(v[0] * v[3]) >> 8, (v[1] * v[3]) >> 8, (v[2] * v[3]) >> 8, v[4], v[0], v[1], v[2], v[5]}
		default:
			return fmt.Errorf("HDR endpoint in LDR ASTC texture")
		}
		if blue {
			e[0], e[1] = (e[0]+e[2])>>1, (e[1]+e[2])>>1
			e[4], e[5] = (e[4]+e[6])>>1, (e[5]+e[6])>>1
		}
		if clamp {
			for i := range e {
				e[i] = astcClamp(e[i])
			}
		}
		d.endpoints[part] = e
	}
	return nil
}

func (d *astcBlock) decodeWeights(buf []byte) {
	a, b := weightA[d.weightRange], weightB[d.weightRange]
	seq := astcSequence(buf, 128, a, b, d.weightCount, true)
	var values [160]int
	for i, s := range seq {
		v := 0
		x := s.bits
		if a == 0 {
			switch b {
			case 1:
				v = x * 63
			case 2:
				v = x<<4 | x<<2 | x
			case 3:
				v = x<<3 | x
			case 4:
				v = x<<2 | x>>2
			case 5:
				v = x<<1 | x>>4
			}
		} else if b == 0 {
			v = s.nonbits * 16
			if a == 3 {
				v = s.nonbits * 32
			}
			values[i] = v
			continue
		} else {
			if a == 3 {
				switch b {
				case 1:
					v = s.nonbits * 50
				case 2:
					v = s.nonbits * 23
					if x&2 != 0 {
						v += 0b1000101
					}
				case 3:
					v = s.nonbits*11 + ((x<<4 | x>>1) & 0b1100011)
				}
			} else {
				v = s.nonbits * 28
				if b == 2 {
					v = s.nonbits * 13
					if x&2 != 0 {
						v += 0b1000010
					}
				}
			}
			mask := (x & 1) * 0x7f
			v = (mask & 0x20) | ((v ^ mask) >> 2)
		}
		if v > 32 {
			v++
		}
		values[i] = v
	}
	ds, dt := (1024+d.bw/2)/(d.bw-1), (1024+d.bh/2)/(d.bh-1)
	for y := range d.bh {
		for x := range d.bw {
			gs, gt := (ds*x*(d.width-1)+32)>>6, (dt*y*(d.height-1)+32)>>6
			fs, ft := gs&15, gt&15
			v := (gs >> 4) + (gt>>4)*d.width
			w11 := (fs*ft + 8) >> 4
			w10, w01, w00 := ft-w11, fs-w11, 16-fs-ft+w11
			for p := range d.planes {
				d.weights[y*d.bw+x][p] = (values[v*d.planes+p]*w00 + values[(v+1)*d.planes+p]*w01 + values[(v+d.width)*d.planes+p]*w10 + values[(v+d.width+1)*d.planes+p]*w11 + 8) >> 4
			}
		}
	}
}

func (d *astcBlock) partitions(buf []byte) {
	seed := (int(binary.LittleEndian.Uint32(buf)) >> 13 & 0x3ff) | ((d.parts - 1) << 10)
	r := uint32(seed)
	r ^= r >> 15
	r -= r << 17
	r += r << 7
	r += r << 4
	r ^= r >> 5
	r += r << 16
	r ^= r >> 7
	r ^= r >> 3
	r ^= r << 6
	r ^= r >> 17
	var seeds [8]int
	shifts := [2]int{5, 5}
	if seed&2 != 0 {
		shifts[0] = 4
	}
	if d.parts == 3 {
		shifts[1] = 6
	}
	for i := range seeds {
		v := int(r>>(i*4)) & 15
		j := 1 - i%2
		if seed&1 != 0 {
			j = i % 2
		}
		seeds[i] = (v * v) >> shifts[j]
	}
	for y := range d.bh {
		for x := range d.bw {
			xx, yy := x, y
			if d.bw*d.bh < 31 {
				xx, yy = x*2, y*2
			}
			scores := [4]int{}
			for p := range d.parts {
				scores[p] = (seeds[p*2]*xx + seeds[p*2+1]*yy + int(r>>(14-p*4))) & 63
			}
			part := 0
			for p := 1; p < d.parts; p++ {
				if scores[p] > scores[part] {
					part = p
				}
			}
			d.partition[y*d.bw+x] = part
		}
	}
}

func decodeASTC(data []byte, width, height, bw, bh int) (*image.NRGBA, error) {
	xBlocks, yBlocks := (width+bw-1)/bw, (height+bh-1)/bh
	if len(data) < xBlocks*yBlocks*16 {
		return nil, fmt.Errorf("truncated ASTC texture")
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for by := range yBlocks {
		for bx := range xBlocks {
			buf := data[(by*xBlocks+bx)*16:][:16]
			var d astcBlock
			d.bw, d.bh = bw, bh
			constant := [4]byte{}
			void := buf[0] == 0xfc && buf[1]&1 == 1
			if void {
				if buf[1]&2 != 0 {
					return nil, fmt.Errorf("HDR block in LDR ASTC texture")
				}
				constant = [4]byte{buf[9], buf[11], buf[13], buf[15]}
			} else {
				if (buf[0]&0xc3 == 0xc0 && buf[1]&1 == 1) || buf[0]&15 == 0 {
					return nil, fmt.Errorf("invalid ASTC block")
				}
				if err := d.parameters(buf); err != nil {
					return nil, err
				}
				if err := d.decodeEndpoints(buf); err != nil {
					return nil, err
				}
				d.decodeWeights(buf)
				if d.parts > 1 {
					d.partitions(buf)
				}
			}
			for y := 0; y < bh && by*bh+y < height; y++ {
				for x := 0; x < bw && bx*bw+x < width; x++ {
					dst := img.Pix[((height-1-by*bh-y)*width+bx*bw+x)*4:]
					if void {
						copy(dst, constant[:])
						continue
					}
					i := y*bw + x
					e := d.endpoints[d.partition[i]]
					for c := range 4 {
						plane := 0
						if d.planes == 2 && d.selector == c {
							plane = 1
						}
						weight := d.weights[i][plane]
						v := (((e[c]*257*(64-weight)+e[c+4]*257*weight+32)>>6)*255 + 32768) / 65536
						dst[c] = byte(v)
					}
				}
			}
		}
	}
	return img, nil
}
