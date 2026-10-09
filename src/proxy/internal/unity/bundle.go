// Adapted from UnityPy/files/BundleFile.py (MIT); see LICENSE.UnityPy.

package unity

import (
	"fmt"

	"github.com/pierrec/lz4/v4"
)

type bundleFile struct {
	name string
	data []byte
}

func readBundle(data []byte) ([]bundleFile, error) {
	r := newReader(data)
	signature := r.cstring()
	version := r.u32()
	r.cstring()
	engine := parseVersion(r.cstring())
	if signature != "UnityFS" && !((signature == "UnityRaw" || signature == "UnityWeb") && version == 6) {
		return nil, fmt.Errorf("unsupported Unity bundle: %s version %d", signature, version)
	}
	if r.u64() != uint64(len(data)) {
		return nil, fmt.Errorf("Unity bundle size mismatch")
	}
	compressedSize, size, flags := r.u32(), r.u32(), r.u32()
	if signature != "UnityFS" {
		r.u8()
	}
	if version >= 7 || (engine[0] == 2019 && engine.atLeast(2019, 4, 15)) {
		r.align(16)
	}
	start := r.pos
	if flags&0x80 != 0 {
		r.seek(len(data) - compressedSize)
	}
	info, err := decompress(r.take(compressedSize), size, flags&0x3f)
	if err != nil {
		return nil, err
	}
	if flags&0x80 != 0 {
		r.seek(start)
	}
	if flags&0x200 != 0 {
		r.align(16)
	}
	blocks := newReader(info)
	blocks.take(16)
	type block struct{ size, compressedSize, flags int }
	entries := make([]block, blocks.count(10))
	total := 0
	for i := range entries {
		entries[i] = block{blocks.u32(), blocks.u32(), blocks.u16()}
		if entries[i].size > maxAssetSize-total {
			return nil, fmt.Errorf("Unity bundle exceeds decode size limit")
		}
		total += entries[i].size
	}
	type node struct {
		offset, size uint64
		name         string
	}
	nodes := make([]node, blocks.count(21))
	for i := range nodes {
		nodes[i].offset, nodes[i].size = blocks.u64(), blocks.u64()
		blocks.u32()
		nodes[i].name = blocks.cstring()
	}
	stream := make([]byte, 0, total)
	for _, b := range entries {
		decoded, err := decompress(r.take(b.compressedSize), b.size, b.flags&0x3f)
		if err != nil {
			return nil, err
		}
		stream = append(stream, decoded...)
	}
	files := make([]bundleFile, len(nodes))
	for i, n := range nodes {
		files[i] = bundleFile{n.name, slice(stream, n.offset, n.size)}
	}
	return files, nil
}

func decompress(data []byte, size, compression int) ([]byte, error) {
	if size < 0 || size > maxAssetSize {
		return nil, fmt.Errorf("invalid Unity block size: %d", size)
	}
	switch compression {
	case 0:
		if len(data) != size {
			return nil, fmt.Errorf("Unity block size mismatch")
		}
		return data, nil
	case 2, 3:
		decoded := make([]byte, size)
		n, err := lz4.UncompressBlock(data, decoded)
		if err != nil {
			return nil, err
		}
		if n != size {
			return nil, fmt.Errorf("Unity LZ4 block size mismatch")
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("unsupported Unity compression: %d", compression)
	}
}
