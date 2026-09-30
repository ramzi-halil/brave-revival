package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"
	"unicode/utf16"
)

const (
	noEntry                 = uint32(0xffffffff)
	androidNS               = "http://schemas.android.com/apk/res/android"
	networkSecurityConfigID = uint32(0x01010527)
	configPath              = "res/xml/brave_revival_network_security_config.xml"
	configName              = "brave_revival_network_security_config"
)

var le = binary.LittleEndian

func u16(b []byte, off int) uint16      { return le.Uint16(b[off:]) }
func u32(b []byte, off int) uint32      { return le.Uint32(b[off:]) }
func put16(b []byte, off int, v uint16) { le.PutUint16(b[off:], v) }
func put32(b []byte, off int, v uint32) { le.PutUint32(b[off:], v) }

func chunk(kind uint16, header, size int) []byte {
	b := make([]byte, size)
	put16(b, 0, kind)
	put16(b, 2, uint16(header))
	put32(b, 4, uint32(size))
	return b
}

func children(b []byte, kind uint16) ([][]byte, error) {
	if len(b) < 8 || u16(b, 0) != kind || int(u32(b, 4)) != len(b) {
		return nil, fmt.Errorf("invalid chunk %#x", kind)
	}
	off := int(u16(b, 2))
	if off < 8 || off > len(b) {
		return nil, fmt.Errorf("invalid header size")
	}
	var out [][]byte
	for off < len(b) {
		if len(b)-off < 8 {
			return nil, fmt.Errorf("truncated chunk header")
		}
		size, header := int(u32(b, off+4)), int(u16(b, off+2))
		if header < 8 || size < header || size > len(b)-off {
			return nil, fmt.Errorf("invalid child chunk size")
		}
		out = append(out, b[off:off+size])
		off += size
	}
	return out, nil
}

func assemble(header []byte, parts [][]byte) []byte {
	out := bytes.Clone(header)
	for _, p := range parts {
		out = append(out, p...)
	}
	put32(out, 4, uint32(len(out)))
	return out
}

func poolStrings(b []byte) ([]string, error) {
	if len(b) < 28 || u16(b, 0) != 1 {
		return nil, fmt.Errorf("invalid string pool")
	}
	n, header, start := int(u32(b, 8)), int(u16(b, 2)), int(u32(b, 20))
	if header < 28 || n > (len(b)-header)/4 || start < header+n*4 || start > len(b) {
		return nil, fmt.Errorf("invalid string offsets")
	}
	var out []string
	for i := 0; i < n; i++ {
		p := start + int(u32(b, header+4*i))
		if p < start || p >= len(b) {
			return nil, fmt.Errorf("string outside pool")
		}
		if u32(b, 16)&256 != 0 {
			readLen := func() (int, error) {
				if p >= len(b) {
					return 0, fmt.Errorf("truncated UTF-8 length")
				}
				v := int(b[p])
				p++
				if v&128 != 0 {
					if p >= len(b) {
						return 0, fmt.Errorf("truncated UTF-8 length")
					}
					v = (v&127)<<8 | int(b[p])
					p++
				}
				return v, nil
			}
			if _, err := readLen(); err != nil {
				return nil, err
			}
			l, err := readLen()
			if err != nil {
				return nil, err
			}
			if l >= len(b)-p || b[p+l] != 0 {
				return nil, fmt.Errorf("truncated UTF-8 string")
			}
			out = append(out, string(b[p:p+l]))
		} else {
			if len(b)-p < 2 {
				return nil, fmt.Errorf("truncated UTF-16 length")
			}
			l := int(u16(b, p))
			p += 2
			if l&0x8000 != 0 {
				if len(b)-p < 2 {
					return nil, fmt.Errorf("truncated UTF-16 length")
				}
				l = (l&0x7fff)<<16 | int(u16(b, p))
				p += 2
			}
			if l >= (len(b)-p)/2 || u16(b, p+2*l) != 0 {
				return nil, fmt.Errorf("truncated UTF-16 string")
			}
			units := make([]uint16, l)
			for j := range units {
				units[j] = u16(b, p+2*j)
			}
			out = append(out, string(utf16.Decode(units)))
		}
	}
	return out, nil
}

// Append without renumbering existing strings, including styled string references.
func appendString(b []byte, s string) ([]byte, uint32, error) {
	if _, err := poolStrings(b); err != nil {
		return nil, 0, err
	}
	n, h, start, styles := u32(b, 8), int(u16(b, 2)), int(u32(b, 20)), int(u32(b, 24))
	end := len(b)
	if styles != 0 {
		if styles < start || styles > len(b) {
			return nil, 0, fmt.Errorf("invalid style offset")
		}
		end = styles
	}
	var encoded []byte
	units := utf16.Encode([]rune(s))
	if u32(b, 16)&256 != 0 {
		if len(units) > 0x7fff || len(s) > 0x7fff {
			return nil, 0, fmt.Errorf("string too long")
		}
		for _, l := range []int{len(units), len(s)} {
			if l > 127 {
				encoded = append(encoded, byte(l>>8)|128)
			}
			encoded = append(encoded, byte(l))
		}
		encoded = append(encoded, s...)
		encoded = append(encoded, 0)
	} else {
		if len(units) > 0x7fff {
			return nil, 0, fmt.Errorf("string too long")
		}
		encoded = le.AppendUint16(encoded, uint16(len(units)))
		for _, v := range units {
			encoded = le.AppendUint16(encoded, v)
		}
		encoded = append(encoded, 0, 0)
	}
	for len(encoded)%4 != 0 {
		encoded = append(encoded, 0)
	}
	pos := h + int(n)*4
	out := bytes.Clone(b[:pos])
	out = le.AppendUint32(out, uint32(end-start))
	out = append(out, b[pos:end]...)
	out = append(out, encoded...)
	out = append(out, b[end:]...)
	put32(out, 4, uint32(len(out)))
	put32(out, 8, n+1)
	put32(out, 16, u32(b, 16)&^1) // Appending invalidates the pool's sorted flag.
	put32(out, 20, uint32(start+4))
	if styles != 0 {
		put32(out, 24, uint32(styles+4+len(encoded)))
	}
	return out, n, nil
}

func addResource(b []byte) ([]byte, uint32, error) {
	parts, err := children(b, 2)
	if err != nil {
		return nil, 0, err
	}
	if len(b) < 12 || u32(b, 8) != 1 || len(parts) != 2 || u16(parts[0], 0) != 1 || u16(parts[1], 0) != 0x200 {
		return nil, 0, fmt.Errorf("expected one resource package")
	}
	var pathIndex uint32
	parts[0], pathIndex, err = appendString(parts[0], configPath)
	if err != nil {
		return nil, 0, err
	}
	pkg := parts[1]
	if len(pkg) < 288 || u16(pkg, 2) != 288 || u32(pkg, 284) != 0 {
		return nil, 0, fmt.Errorf("unsupported resource package header/typeIdOffset")
	}
	pc, err := children(pkg, 0x200)
	if err != nil {
		return nil, 0, err
	}
	if len(pc) < 2 || u16(pc[0], 0) != 1 || u16(pc[1], 0) != 1 || u32(pkg, 268) != 288 || int(u32(pkg, 276)) != 288+len(pc[0]) {
		return nil, 0, fmt.Errorf("unsupported package string pool layout")
	}
	names, err := poolStrings(pc[0])
	if err != nil {
		return nil, 0, err
	}
	xmlType := slices.Index(names, "xml") + 1
	if xmlType == 0 {
		return nil, 0, fmt.Errorf("package has no xml resource type")
	}
	keys, err := poolStrings(pc[1])
	if err != nil {
		return nil, 0, err
	}
	if slices.Contains(keys, configName) {
		return nil, 0, fmt.Errorf("APK is already patched")
	}
	pc[1], _, err = appendString(pc[1], configName)
	if err != nil {
		return nil, 0, err
	}
	entryIndex := -1
	for i, c := range pc {
		if u16(c, 0) != 0x202 || len(c) < 16 || int(c[8]) != xmlType {
			continue
		}
		entryIndex = int(u32(c, 12))
		if entryIndex >= 65535 || len(c) != int(u16(c, 2))+4*entryIndex {
			return nil, 0, fmt.Errorf("invalid xml type spec")
		}
		c = bytes.Clone(c)
		c = le.AppendUint32(c, 0)
		put32(c, 12, uint32(entryIndex+1))
		put32(c, 4, uint32(len(c)))
		pc[i] = c
	}
	if entryIndex < 0 {
		return nil, 0, fmt.Errorf("missing xml type spec")
	}
	defaultFound := false
	for i, c := range pc {
		if u16(c, 0) != 0x201 || len(c) < 24 || int(c[8]) != xmlType {
			continue
		}
		h, count, start := int(u16(c, 2)), int(u32(c, 12)), int(u32(c, 16))
		if c[9] != 0 || count > entryIndex || h < 24 || start < h+4*count || start > len(c) {
			return nil, 0, fmt.Errorf("unsupported sparse/offset16 xml resource table")
		}
		isDefault := h >= 24 && bytes.Equal(c[24:h], make([]byte, h-24))
		if isDefault && defaultFound {
			return nil, 0, fmt.Errorf("multiple default xml configurations")
		}
		out := bytes.Clone(c[:h+4*count])
		for j := count; j <= entryIndex; j++ {
			out = le.AppendUint32(out, noEntry)
		}
		delta := 4 * (entryIndex + 1 - count)
		out = append(out, c[h+4*count:]...)
		put32(out, 12, uint32(entryIndex+1))
		put32(out, 16, uint32(start+delta))
		if isDefault {
			defaultFound = true
			put32(out, h+4*entryIndex, uint32(len(c)-start))
			entry := make([]byte, 16)
			put16(entry, 0, 8)
			put32(entry, 4, uint32(len(keys)))
			put16(entry, 8, 8)
			entry[11] = 3
			put32(entry, 12, pathIndex)
			out = append(out, entry...)
		}
		put32(out, 4, uint32(len(out)))
		pc[i] = out
	}
	if !defaultFound {
		return nil, 0, fmt.Errorf("no default xml configuration")
	}
	parts[1] = assemble(pkg[:288], pc)
	return assemble(b[:int(u16(b, 2))], parts), u32(pkg, 8)<<24 | uint32(xmlType)<<16 | uint32(entryIndex), nil
}

func patchManifest(b []byte, resID uint32) ([]byte, error) {
	parts, err := children(b, 3)
	if err != nil {
		return nil, err
	}
	if len(parts) < 2 || u16(parts[0], 0) != 1 || u16(parts[1], 0) != 0x180 {
		return nil, fmt.Errorf("missing manifest string pool/resource map")
	}
	strings, err := poolStrings(parts[0])
	if err != nil {
		return nil, err
	}
	ns := slices.Index(strings, androidNS)
	if ns < 0 {
		return nil, fmt.Errorf("missing Android namespace")
	}
	var name uint32
	parts[0], name, err = appendString(parts[0], "networkSecurityConfig")
	if err != nil {
		return nil, err
	}
	resourceMap := parts[1]
	if len(resourceMap) > 8+4*int(name) {
		return nil, fmt.Errorf("invalid manifest resource map")
	}
	m := make([]byte, 8+4*(int(name)+1))
	copy(m, resourceMap)
	put32(m, 4, uint32(len(m)))
	put32(m, 8+4*int(name), networkSecurityConfigID)
	parts[1] = m
	patched := false
	for i, c := range parts {
		if u16(c, 0) != 0x102 {
			continue
		}
		h := int(u16(c, 2))
		if len(c) < h+20 {
			return nil, fmt.Errorf("truncated XML element")
		}
		n := int(u32(c, h+4))
		if n >= len(strings) || strings[n] != "application" {
			continue
		}
		if patched {
			return nil, fmt.Errorf("multiple application elements")
		}
		patched = true
		start, size, count := h+int(u16(c, h+8)), int(u16(c, h+10)), int(u16(c, h+12))
		if size != 20 || start < h+20 || start+size*count != len(c) {
			return nil, fmt.Errorf("unsupported application attributes")
		}
		insert := count
		for j := 0; j < count; j++ {
			a := c[start+20*j:]
			ref := u32(a, 4)
			id := uint32(0)
			if uint64(ref)*4+12 <= uint64(len(m)) {
				id = u32(m, 8+4*int(ref))
			}
			if id == networkSecurityConfigID {
				return nil, fmt.Errorf("existing networkSecurityConfig; refusing to replace policy")
			}
			if id > networkSecurityConfigID && insert == count {
				insert = j
			}
		}
		a := make([]byte, 20)
		put32(a, 0, uint32(ns))
		put32(a, 4, name)
		put32(a, 8, noEntry)
		put16(a, 12, 8)
		a[15] = 1
		put32(a, 16, resID)
		out := bytes.Clone(c[:start+20*insert])
		out = append(out, a...)
		out = append(out, c[start+20*insert:]...)
		put16(out, h+12, uint16(count+1))
		for _, p := range []int{h + 14, h + 16, h + 18} {
			v := u16(out, p)
			if int(v) > insert {
				put16(out, p, v+1)
			}
		}
		put32(out, 4, uint32(len(out)))
		parts[i] = out
	}
	if !patched {
		return nil, fmt.Errorf("missing application element")
	}
	return assemble(b[:int(u16(b, 2))], parts), nil
}

func networkConfig() []byte {
	pool := chunk(1, 28, 28)
	put32(pool, 16, 256)
	put32(pool, 20, 28)
	for _, s := range []string{"network-security-config", "base-config", "trust-anchors", "certificates", "src", "user", "system"} {
		pool, _, _ = appendString(pool, s)
	}
	parts := [][]byte{pool}
	start := func(name uint32, value uint32) {
		size := 36
		if value != noEntry {
			size += 20
		}
		b := chunk(0x102, 16, size)
		put32(b, 8, 1)
		put32(b, 12, noEntry)
		put32(b, 16, noEntry)
		put32(b, 20, name)
		put16(b, 24, 20)
		put16(b, 26, 20)
		if value != noEntry {
			put16(b, 28, 1)
			put32(b, 36, noEntry)
			put32(b, 40, 4)
			put32(b, 44, value)
			put16(b, 48, 8)
			b[51] = 3
			put32(b, 52, value)
		}
		parts = append(parts, b)
	}
	end := func(name uint32) {
		b := chunk(0x103, 16, 24)
		put32(b, 8, 1)
		put32(b, 12, noEntry)
		put32(b, 16, noEntry)
		put32(b, 20, name)
		parts = append(parts, b)
	}
	start(0, noEntry)
	start(1, noEntry)
	start(2, noEntry)
	start(3, 5)
	end(3)
	start(3, 6)
	end(3)
	end(2)
	end(1)
	end(0)
	return assemble(chunk(3, 8, 8), parts)
}
