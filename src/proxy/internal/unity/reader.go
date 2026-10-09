// Adapted from UnityPy (MIT); see LICENSE.UnityPy.

package unity

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const maxAssetSize = 512 << 20

type reader struct {
	data  []byte
	pos   int
	order binary.ByteOrder
}

func newReader(data []byte) *reader {
	return &reader{data: data, order: binary.BigEndian}
}

// Bounds failures unwind to TexturePNG so binary layouts can be read without losing error context.
func (r *reader) take(n int) []byte {
	if n < 0 || n > len(r.data)-r.pos {
		panic(fmt.Errorf("truncated Unity asset at offset %d (need %d bytes)", r.pos, n))
	}
	b := r.data[r.pos : r.pos+n]
	r.pos += n
	return b
}

func (r *reader) seek(pos int) {
	if pos < 0 || pos > len(r.data) {
		panic(fmt.Errorf("Unity offset outside asset: %d", pos))
	}
	r.pos = pos
}

func (r *reader) align(n int) { r.seek((r.pos + n - 1) & -n) }
func (r *reader) u8() int     { return int(r.take(1)[0]) }
func (r *reader) u16() int    { return int(r.order.Uint16(r.take(2))) }
func (r *reader) u32() int    { return int(r.order.Uint32(r.take(4))) }
func (r *reader) i32() int    { return int(int32(r.u32())) }
func (r *reader) u64() uint64 { return r.order.Uint64(r.take(8)) }

func (r *reader) count(minSize int) int {
	n := r.i32()
	if n < 0 || n > (len(r.data)-r.pos)/minSize {
		panic(fmt.Errorf("invalid Unity array size: %d", n))
	}
	return n
}

func (r *reader) cstring() string {
	n := bytes.IndexByte(r.data[r.pos:], 0)
	if n < 0 {
		panic(fmt.Errorf("unterminated Unity string"))
	}
	return string(r.take(n + 1)[:n])
}

func (r *reader) byteArray() []byte { return r.take(r.count(1)) }

func (r *reader) string() string {
	s := string(r.byteArray())
	r.align(4)
	return s
}

func slice(data []byte, offset, size uint64) []byte {
	if offset > uint64(len(data)) || size > uint64(len(data))-offset {
		panic(fmt.Errorf("Unity data outside asset"))
	}
	return data[offset : offset+size]
}
