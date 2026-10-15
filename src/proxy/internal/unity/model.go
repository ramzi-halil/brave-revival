// Adapted from UnityPy/helpers/MeshHelper.py and export/MeshExporter.py.

package unity

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"path"
	"slices"
)

type Mesh struct {
	Name           string        `json:"name"`
	Geometry       Geometry      `json:"geometry"`
	BindPoses      [][16]float32 `json:"bindPoses,omitempty"`
	BoneNameHashes []uint32      `json:"boneNameHashes,omitempty"`
}

type Geometry struct {
	Data GeometryData `json:"data"`
}

type GeometryData struct {
	Attributes map[string]Attribute `json:"attributes"`
	Index      Index                `json:"index"`
}

type Attribute struct {
	ItemSize int       `json:"itemSize"`
	Type     string    `json:"type"`
	Array    []float32 `json:"array"`
}

type Index struct {
	Type  string   `json:"type"`
	Array []uint32 `json:"array"`
}

func ModelMeshes(data []byte, name string, names ...string) (meshes []Mesh, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			meshes = nil
			err = fmt.Errorf("failed to decode Unity mesh: %v", failure)
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
		err = readAssetObjects(file.data, func(typ assetType, object *reader, engine unityVersion, _ int, _ int64) bool {
			if typ.classID != 43 {
				return false
			}
			if len(typ.nodes) == 0 {
				panic(fmt.Errorf("Unity Mesh has no type tree"))
			}
			values := readTreeValue(object, typ.nodes, 0).(map[string]any)
			if len(names) != 0 && !slices.Contains(names, values["m_Name"].(string)) {
				return false
			}
			meshes = append(meshes, meshFromValues(values, object.order, engine, streams))
			return false
		})
		if err != nil {
			return nil, err
		}
	}
	if len(meshes) == 0 {
		return nil, fmt.Errorf("asset contains no Mesh")
	}
	return meshes, nil
}

func meshFromValues(values map[string]any, order binary.ByteOrder, version unityVersion, streams map[string][]byte) Mesh {
	number := func(values map[string]any, key string) int {
		value, _ := values[key].(int64)
		return int(value)
	}
	if compressed, ok := values["m_CompressedMesh"].(map[string]any); ok {
		for _, field := range []string{"m_Vertices", "m_UV", "m_Normals", "m_Triangles"} {
			packed, _ := compressed[field].(map[string]any)
			if number(packed, "m_NumItems") != 0 {
				panic(fmt.Errorf("unsupported packed Unity mesh: %s", field))
			}
		}
	}
	vertices := values["m_VertexData"].(map[string]any)
	count := number(vertices, "m_VertexCount")
	if count <= 0 || count > maxAssetSize/12 {
		panic(fmt.Errorf("invalid Unity vertex count: %d", count))
	}
	data, _ := vertices["m_DataSize"].([]byte)
	if info, ok := values["m_StreamData"].(map[string]any); ok {
		if name, _ := info["path"].(string); name != "" {
			data = streamData(name, uint64(number(info, "offset")), uint64(number(info, "size")), streams)
		}
	}
	type channel struct{ stream, offset, format, dimension int }
	channels := []channel{}
	var strides, offsets [16]int
	for _, raw := range vertices["m_Channels"].([]any) {
		value := raw.(map[string]any)
		c := channel{number(value, "stream"), number(value, "offset"), number(value, "format"), number(value, "dimension") & 15}
		if c.dimension != 0 {
			if c.stream < 0 || c.stream >= len(strides) || c.dimension > 4 {
				panic(fmt.Errorf("invalid Unity vertex channel"))
			}
			c.format = vertexFormat(c.format, version)
			strides[c.stream] += c.dimension * componentSize(c.format)
		}
		channels = append(channels, c)
	}
	for i := 1; i < len(offsets); i++ {
		offsets[i] = (offsets[i-1] + count*strides[i-1] + 15) &^ 15
	}
	for i, stride := range strides {
		if stride != 0 && count*stride > len(data)-offsets[i] {
			panic(fmt.Errorf("truncated Unity vertex stream"))
		}
	}
	attributes := make(map[string]Attribute)
	uvChannel := 4
	if !version.atLeast(2018, 0, 0) {
		uvChannel = 3
	}
	for _, target := range []struct {
		index, size int
		name        string
	}{{0, 3, "position"}, {1, 3, "normal"}, {uvChannel, 2, "uv"}, {12, 4, "skinWeight"}, {13, 4, "skinIndex"}} {
		if target.index >= 12 && !version.atLeast(2018, 2, 0) {
			continue
		}
		if target.index >= len(channels) || channels[target.index].dimension == 0 {
			continue
		}
		c := channels[target.index]
		size := componentSize(c.format)
		if c.dimension < target.size || c.offset < 0 || c.offset+c.dimension*size > strides[c.stream] {
			panic(fmt.Errorf("invalid Unity %s channel", target.name))
		}
		array := make([]float32, count*target.size)
		for vertex := range count {
			start := offsets[c.stream] + vertex*strides[c.stream] + c.offset
			r := newReader(slice(data, uint64(start), uint64(target.size*size)))
			r.order = order
			for dimension := range target.size {
				value := vertexComponent(r, c.format)
				if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
					panic(fmt.Errorf("non-finite Unity vertex attribute"))
				}
				// Mirror X together with triangle winding to convert Unity's handedness.
				if dimension == 0 && (target.name == "position" || target.name == "normal") {
					value = -value
				}
				array[vertex*target.size+dimension] = value
			}
		}
		attributes[target.name] = Attribute{target.size, "Float32Array", array}
	}
	mesh := Mesh{Name: values["m_Name"].(string)}
	weights, hasWeights := attributes["skinWeight"]
	boneIndices, hasIndices := attributes["skinIndex"]
	if hasWeights != hasIndices {
		panic(fmt.Errorf("Unity mesh has incomplete skinning attributes"))
	}
	if hasWeights {
		poses := values["m_BindPose"].([]any)
		hashes := values["m_BoneNameHashes"].([]any)
		if len(poses) == 0 || len(poses) != len(hashes) || len(poses) > 65536 {
			panic(fmt.Errorf("invalid Unity bone palette"))
		}
		mesh.BindPoses = make([][16]float32, len(poses))
		mesh.BoneNameHashes = make([]uint32, len(hashes))
		for i, raw := range poses {
			matrix := raw.(map[string]any)
			for row := range 4 {
				for column := range 4 {
					value := math.Float32frombits(uint32(matrix[fmt.Sprintf("e%d%d", row, column)].(int64)))
					if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
						panic(fmt.Errorf("non-finite Unity bind pose"))
					}
					// Change the inverse bind's basis with H * M * H, where H mirrors X.
					if (row == 0) != (column == 0) {
						value = -value
					}
					mesh.BindPoses[i][column*4+row] = value
				}
			}
			mesh.BoneNameHashes[i] = uint32(hashes[i].(int64))
		}
		for vertex := range count {
			var sum float32
			for influence := range 4 {
				i := vertex*4 + influence
				weight, index := weights.Array[i], boneIndices.Array[i]
				if weight < 0 || index < 0 || index != float32(int(index)) || index >= float32(len(poses)) {
					panic(fmt.Errorf("invalid Unity bone influence"))
				}
				sum += weight
			}
			if sum <= 0 || math.IsInf(float64(sum), 0) {
				panic(fmt.Errorf("invalid Unity skin weight sum"))
			}
		}
		boneIndices.Type = "Uint16Array"
		attributes["skinIndex"] = boneIndices
	}
	if _, ok := attributes["position"]; !ok {
		panic(fmt.Errorf("Unity mesh has no positions"))
	}
	if _, ok := attributes["uv"]; !ok {
		panic(fmt.Errorf("Unity mesh has no UV coordinates"))
	}
	indexSize := 2
	if format := number(values, "m_IndexFormat"); format == 1 {
		indexSize = 4
	} else if format != 0 {
		panic(fmt.Errorf("invalid Unity index format"))
	}
	indexBuffer := values["m_IndexBuffer"].([]byte)
	indices := []uint32{}
	for _, raw := range values["m_SubMeshes"].([]any) {
		submesh := raw.(map[string]any)
		start, n, base := number(submesh, "firstByte"), number(submesh, "indexCount"), number(submesh, "baseVertex")
		if start < 0 || start%indexSize != 0 || n < 0 || n > len(indexBuffer)/indexSize {
			panic(fmt.Errorf("invalid Unity submesh indices"))
		}
		r := newReader(slice(indexBuffer, uint64(start), uint64(n*indexSize)))
		r.order = order
		part := make([]uint32, n)
		for i := range part {
			var index int
			if indexSize == 4 {
				index = r.u32()
			} else {
				index = r.u16()
			}
			index += base
			if index < 0 || index >= count {
				panic(fmt.Errorf("Unity vertex index outside mesh"))
			}
			part[i] = uint32(index)
		}
		triangle := func(a, b, c uint32) { indices = append(indices, c, b, a) }
		switch number(submesh, "topology") {
		case 0:
			if n%3 != 0 {
				panic(fmt.Errorf("invalid Unity triangle count"))
			}
			for i := 0; i < n; i += 3 {
				triangle(part[i], part[i+1], part[i+2])
			}
		case 1:
			for i := 0; i+2 < n; i++ {
				a, b, c := part[i], part[i+1], part[i+2]
				if a == b || b == c || a == c {
					continue
				}
				if i%2 != 0 {
					a, b = b, a
				}
				triangle(a, b, c)
			}
		case 2:
			if n%4 != 0 {
				panic(fmt.Errorf("invalid Unity quad count"))
			}
			for i := 0; i < n; i += 4 {
				triangle(part[i], part[i+1], part[i+2])
				triangle(part[i], part[i+2], part[i+3])
			}
		default:
			panic(fmt.Errorf("unsupported Unity mesh topology"))
		}
	}
	mesh.Geometry = Geometry{GeometryData{attributes, Index{"Uint32Array", indices}}}
	return mesh
}

func vertexFormat(format int, version unityVersion) int {
	if version.atLeast(2019, 0, 0) {
		return format
	}
	if version.atLeast(2017, 0, 0) {
		if format > 2 {
			return format - 1
		}
		return format
	}
	return [...]int{0, 1, 2, 6, 10}[format]
}

func componentSize(format int) int {
	if format < 0 || format > 11 {
		panic(fmt.Errorf("unsupported Unity vertex format: %d", format))
	}
	return [...]int{4, 2, 1, 1, 2, 2, 1, 1, 2, 2, 4, 4}[format]
}

func vertexComponent(r *reader, format int) float32 {
	switch format {
	case 0:
		return math.Float32frombits(uint32(r.u32()))
	case 1:
		bits := r.u16()
		exponent, fraction := (bits>>10)&31, bits&1023
		value := math.Ldexp(float64(fraction), -24)
		if exponent == 31 {
			panic(fmt.Errorf("non-finite Unity half-float"))
		} else if exponent != 0 {
			value = math.Ldexp(float64(1024+fraction), exponent-25)
		}
		if bits&0x8000 != 0 {
			value = -value
		}
		return float32(value)
	case 2:
		return float32(r.u8()) / 255
	case 3:
		return max(-1, float32(int8(r.u8()))/127)
	case 4:
		return float32(r.u16()) / 65535
	case 5:
		return max(-1, float32(int16(r.u16()))/32767)
	case 6:
		return float32(r.u8())
	case 7:
		return float32(int8(r.u8()))
	case 8:
		return float32(r.u16())
	case 9:
		return float32(int16(r.u16()))
	case 10:
		return float32(r.u32())
	case 11:
		return float32(r.i32())
	default:
		panic(fmt.Errorf("unsupported Unity vertex format: %d", format))
	}
}
