// Adapted from UnityPy/helpers/TypeTreeNode.py and TypeTreeHelper.py.

package unity

import "fmt"

type typeNode struct {
	level, size, flags, end int
	typ, name               string
}

func readTypeTree(r *reader, version int) []typeNode {
	n := r.count(24)
	stringSize := r.i32()
	nodeSize := 24
	if version >= 19 {
		nodeSize += 8
	}
	raw := newReader(r.take(n * nodeSize))
	raw.order = r.order
	strings := r.take(stringSize)
	getString := func(offset int) string {
		if offset&0x80000000 != 0 {
			return commonStrings[offset&0x7fffffff]
		}
		s := newReader(strings)
		s.seek(offset)
		return s.cstring()
	}
	nodes := make([]typeNode, n)
	for i := range nodes {
		raw.u16()
		level := raw.u8()
		raw.u8()
		typ, name := getString(raw.u32()), getString(raw.u32())
		size := raw.i32()
		raw.u32()
		flags := raw.u32()
		if version >= 19 {
			raw.u64()
		}
		nodes[i] = typeNode{level, size, flags, n, typ, name}
		if (i == 0 && level != 0) || (i > 0 && (level == 0 || level > nodes[i-1].level+1)) {
			panic(fmt.Errorf("invalid Unity type-tree level"))
		}
	}
	stack := []int{}
	for i := range nodes {
		for len(stack) != 0 && nodes[stack[len(stack)-1]].level >= nodes[i].level {
			nodes[stack[len(stack)-1]].end = i
			stack = stack[:len(stack)-1]
		}
		stack = append(stack, i)
	}
	return nodes
}

func readTreeValue(r *reader, nodes []typeNode, index int) (value any) {
	node := nodes[index]
	defer func() {
		if node.flags&0x4000 != 0 {
			r.align(4)
		}
	}()
	switch node.typ {
	case "string":
		return r.string()
	case "TypelessData":
		return r.byteArray()
	case "SInt8", "char":
		return int64(int8(r.u8()))
	case "UInt8", "bool":
		return int64(r.u8())
	case "short", "SInt16":
		return int64(int16(r.u16()))
	case "unsigned short", "UInt16":
		return int64(r.u16())
	case "int", "SInt32":
		return int64(r.i32())
	case "unsigned int", "UInt32", "float":
		return int64(r.u32())
	case "long long", "SInt64", "UInt64", "unsigned long long", "double":
		return int64(r.u64())
	case "Array":
		if index+2 >= node.end {
			panic(fmt.Errorf("invalid Unity array type tree"))
		}
		element := index + 2
		n := r.count(max(1, nodes[element].size))
		if nodes[element].size == 1 && nodes[element].end == element+1 {
			return r.take(n)
		}
		for range n {
			readTreeValue(r, nodes, element)
		}
		return nil
	}
	if index+1 < node.end && nodes[index+1].typ == "Array" {
		return readTreeValue(r, nodes, index+1)
	}
	if node.end == index+1 {
		return r.take(node.size)
	}
	fields := make(map[string]any)
	for child := index + 1; child < node.end; child = nodes[child].end {
		fields[nodes[child].name] = readTreeValue(r, nodes, child)
	}
	return fields
}
