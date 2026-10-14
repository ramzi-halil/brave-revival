package unity

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelMeshes(t *testing.T) {
	data, err := os.ReadFile("testdata/model.unity3d")
	require.NoError(t, err)
	meshes, err := ModelMeshes(data, "model")
	require.NoError(t, err)
	require.Len(t, meshes, 3)
	names := []string{}
	for _, mesh := range meshes {
		names = append(names, mesh.Name)
		geometry := mesh.Geometry.Data
		require.Equal(t, Attribute{3, "Float32Array", []float32{0, 0, 0, -1, 2, 3, -4, 5, 6, -7, 8, 9}}, geometry.Attributes["position"])
		require.Equal(t, Attribute{3, "Float32Array", []float32{0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0}}, geometry.Attributes["normal"])
		require.Equal(t, Attribute{2, "Float32Array", []float32{0, 1, .25, .75, .5, .5, .75, .25}}, geometry.Attributes["uv"])
		require.Equal(t, Index{"Uint32Array", []uint32{3, 2, 1}}, geometry.Index)
	}
	require.ElementsMatch(t, []string{"lh", "rh", "f"}, names)
	filtered, err := ModelMeshes(data, "model", "lh", "rh")
	require.NoError(t, err)
	require.Len(t, filtered, 2)
	require.ElementsMatch(t, []string{"lh", "rh"}, []string{filtered[0].Name, filtered[1].Name})
	_, err = ModelMeshes(data, "model", "b")
	require.EqualError(t, err, "asset contains no Mesh")
	_, err = TexturePNG(data, "model")
	require.NoError(t, err)
	for _, broken := range [][]byte{nil, []byte("invalid"), []byte("UnityFS\x00"), data[:len(data)-1]} {
		_, err = ModelMeshes(broken, "broken")
		require.Error(t, err)
	}
	data, err = os.ReadFile("testdata/texture.unity3d")
	require.NoError(t, err)
	_, err = ModelMeshes(data, "texture")
	require.EqualError(t, err, "asset contains no Mesh")
}

func FuzzModelMeshes(f *testing.F) {
	for _, name := range []string{"model.unity3d", "texture.assets", "tree-be.assets"} {
		data, err := os.ReadFile("testdata/" + name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) { ModelMeshes(data, "test") })
}
