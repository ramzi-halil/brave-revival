// Adapted from UnityPy/files/SerializedFile.py and ObjectReader.py (MIT); see LICENSE.UnityPy.

package unity

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

type unityVersion [3]int

func parseVersion(s string) (v unityVersion) {
	parts := strings.FieldsFunc(s, func(c rune) bool { return c < '0' || c > '9' })
	for i := 0; i < min(len(parts), len(v)); i++ {
		v[i], _ = strconv.Atoi(parts[i])
	}
	return
}

func (v unityVersion) atLeast(major, minor, patch int) bool {
	return v[0] > major || (v[0] == major && (v[1] > minor || (v[1] == minor && v[2] >= patch)))
}

func isSerialized(data []byte) bool {
	if len(data) < 20 {
		return false
	}
	v := binary.BigEndian.Uint32(data[8:])
	size := uint64(binary.BigEndian.Uint32(data[4:]))
	if v >= 22 {
		if len(data) < 48 {
			return false
		}
		size = binary.BigEndian.Uint64(data[24:])
	}
	return v >= 13 && v <= 22 && size == uint64(len(data))
}

type assetType struct {
	classID int
	nodes   []typeNode
}

func readAsset(data []byte, streams map[string][]byte) (*texture, error) {
	r := newReader(data)
	r.u32()
	size, version, offset := uint64(r.u32()), r.u32(), uint64(r.u32())
	endian := r.u8()
	r.take(3)
	if version >= 22 {
		r.u32()
		size, offset = r.u64(), r.u64()
		r.u64()
	}
	if version < 13 || version > 22 || size != uint64(len(data)) || offset > size {
		return nil, fmt.Errorf("unsupported or invalid serialized Unity asset")
	}
	if endian == 0 {
		r.order = binary.LittleEndian
	} else if endian != 1 {
		return nil, fmt.Errorf("invalid Unity byte order")
	}
	engine := parseVersion(r.cstring())
	platform, hasTree := r.i32(), r.u8() != 0
	types := make([]assetType, r.count(20))
	for i := range types {
		class := r.i32()
		if version >= 16 {
			r.u8()
		}
		if version >= 17 {
			r.u16()
		}
		if (version < 16 && class < 0) || (version >= 16 && class == 114) {
			r.take(16)
		}
		r.take(16)
		types[i].classID = class
		if hasTree {
			types[i].nodes = readTypeTree(r, version)
			if version >= 21 {
				r.take(r.count(4) * 4)
			}
		}
	}
	bigID := false
	if version < 14 {
		bigID = r.u32() != 0
	}
	count := r.count(16)
	for range count {
		if version >= 14 {
			r.align(4)
		}
		if bigID || version >= 14 {
			r.u64()
		} else {
			r.u32()
		}
		var start uint64
		if version >= 22 {
			start = r.u64()
		} else {
			start = uint64(r.u32())
		}
		length, typeID := r.u32(), r.i32()
		var typ assetType
		if version < 16 {
			typ.classID = r.u16()
			for _, t := range types {
				if t.classID == typeID {
					typ.nodes = t.nodes
					break
				}
			}
		} else {
			if typeID < 0 || typeID >= len(types) {
				return nil, fmt.Errorf("invalid Unity object type index")
			}
			typ = types[typeID]
		}
		if version < 17 {
			r.u16()
		}
		if version == 15 || version == 16 {
			r.u8()
		}
		if typ.classID != 28 {
			continue
		}
		if start > size-offset {
			return nil, fmt.Errorf("Unity object outside asset")
		}
		object := newReader(slice(data, start+offset, uint64(length)))
		object.order = r.order
		if len(typ.nodes) != 0 {
			values := readTreeValue(object, typ.nodes, 0).(map[string]any)
			return textureFromValues(values, streams), nil
		}
		return readTexture(object, engine, platform, streams), nil
	}
	return nil, nil
}
