package proxy

import (
	"crypto/tls"
	"encoding/json"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proxy/internal/unity"
	"example.com/brave-revival/src/www"
	"github.com/stretchr/testify/require"
)

func TestTextureViewer(t *testing.T) {
	fixture, err := os.ReadFile("internal/unity/testdata/texture.unity3d")
	require.NoError(t, err)
	assetsDir := t.TempDir()
	dbDir := t.TempDir()
	fields := (&pmaster.All{}).ProtoReflect().Descriptor().Fields()
	for i := range fields.Len() {
		field := fields.Get(i)
		rowFields := field.Message().Fields()
		columns := make([]string, rowFields.Len())
		for j := range rowFields.Len() {
			columns[j] = string(rowFields.Get(j).Name())
		}
		content := strings.Join(columns, ",") + "\n"
		if field.Name() == "version" {
			content += "ios,1.43.274,6749,12810\n"
		} else if field.Name() == "equipment_character" {
			content += "1,4,0,0,0,0,0,0,0,0,0,0,0\n"
		}
		require.NoError(t, os.WriteFile(filepath.Join(dbDir, string(field.Name())+".csv"), []byte(content), 0o600))
	}
	resources := "id,ios,android,ios_size,android_size\n1,ios-hash,android-hash,440,0\n2,missing-hash,android-hash,0,0\n3,broken-hash,android-hash,0,0\n4,model-hash,model-android-hash,0,0\n5,animation-hash,animation-android-hash,0,0\n"
	require.NoError(t, os.WriteFile(filepath.Join(dbDir, "resources.csv"), []byte(resources), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "broken-hash"), []byte("UnityFS\x00"), 0o600))
	wwwHandler, err := www.NewHandler(&config.Config{DBDir: dbDir, AssetsDir: []string{assetsDir}})
	require.NoError(t, err)
	h := newHandler(&config.Config{}, tls.Certificate{})
	h.initializeWWWHandler(wwwHandler)

	for _, path := range []string{"io/ios-hash.unity3d", "ios-hash", "an/android-hash.unity3d", "android-hash"} {
		t.Run(path, func(t *testing.T) {
			assetPath := filepath.Join(assetsDir, path)
			require.NoError(t, os.MkdirAll(filepath.Dir(assetPath), 0o700))
			require.NoError(t, os.WriteFile(assetPath, fixture, 0o600))
			defer os.Remove(assetPath)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/res/t/1", nil))
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Equal(t, "image/png", w.Header().Get("Content-Type"))
			img, err := png.Decode(w.Body)
			require.NoError(t, err)
			require.Equal(t, 2, img.Bounds().Dx())
			require.Equal(t, 2, img.Bounds().Dy())
			require.Equal(t, color.NRGBA{R: 255, A: 255}, color.NRGBAModel.Convert(img.At(0, 0)))
			require.Equal(t, color.NRGBA{R: 255, G: 255, B: 255, A: 128}, color.NRGBAModel.Convert(img.At(1, 1)))
		})
	}
	for url, status := range map[string]int{
		"/res/t/invalid":    http.StatusBadRequest,
		"/res/t/-1":         http.StatusBadRequest,
		"/res/t/4294967296": http.StatusBadRequest,
		"/res/t/999":        http.StatusNotFound,
		"/res/t/2":          http.StatusNotFound,
		"/res/t/3":          http.StatusUnprocessableEntity,
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		require.Equal(t, status, w.Code, url)
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, strings.Replace(url, "/res/t/", "/res/m/", 1)+"?name=lh", nil))
		require.Equal(t, status, w.Code, url)
	}
	model, err := os.ReadFile("internal/unity/testdata/model.unity3d")
	require.NoError(t, err)
	for _, hash := range []string{"model-hash", "model-android-hash"} {
		t.Run(hash, func(t *testing.T) {
			assetPath := filepath.Join(assetsDir, hash)
			require.NoError(t, os.WriteFile(assetPath, model, 0o600))
			defer os.Remove(assetPath)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/res/m/4?name=lh&name=rh", nil))
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			require.Equal(t, "application/json", w.Header().Get("Content-Type"))
			var reply struct {
				Meshes  []unity.Mesh `json:"meshes"`
				Texture string       `json:"texture"`
			}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &reply))
			require.Len(t, reply.Meshes, 2)
			require.ElementsMatch(t, []string{"lh", "rh"}, []string{reply.Meshes[0].Name, reply.Meshes[1].Name})
			require.Equal(t, []uint32{3, 2, 1}, reply.Meshes[0].Geometry.Data.Index.Array)
			require.Equal(t, "/res/t/4", reply.Texture)
			require.Len(t, reply.Meshes[0].BindPoses, 2)
			require.Equal(t, []uint32{385153371, 1509296682}, reply.Meshes[0].BoneNameHashes)
			require.Equal(t, "Uint16Array", reply.Meshes[0].Geometry.Data.Attributes["skinIndex"].Type)
			require.Equal(t, 4, reply.Meshes[0].Geometry.Data.Attributes["skinWeight"].ItemSize)
		})
	}
	animation, err := os.ReadFile("internal/unity/testdata/animation.unity3d")
	require.NoError(t, err)
	for _, hash := range []string{"animation-hash", "animation-android-hash"} {
		t.Run(hash, func(t *testing.T) {
			assetPath := filepath.Join(assetsDir, hash)
			require.NoError(t, os.WriteFile(assetPath, animation, 0o600))
			defer os.Remove(assetPath)
			for _, container := range []string{"first", "second", "character"} {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/res/a/5/"+container+"/101", nil))
				require.Equal(t, http.StatusOK, w.Code, w.Body.String())
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))
				var reply unity.Animation
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &reply))
				require.Equal(t, "101", reply.Name)
				require.Equal(t, 1.0, reply.Duration)
				require.Len(t, reply.Tracks, 3)
				require.Equal(t, "1509296682.position", reply.Tracks[0].Name)
				require.Equal(t, []float32{0, .25, .5, .75, 1}, reply.Tracks[0].Times)
			}
			for url, status := range map[string]int{
				"/res/a/invalid/first/101":  http.StatusBadRequest,
				"/res/a/999/first/101":      http.StatusNotFound,
				"/res/a/3/first/101":        http.StatusUnprocessableEntity,
				"/res/a/5/missing/101":      http.StatusNotFound,
				"/res/a/5/first/10":         http.StatusNotFound,
				"/res/a/5/test/101":         http.StatusUnprocessableEntity,
				"/res/a/invalid":            http.StatusBadRequest,
				"/res/a/2":                  http.StatusNotFound,
				"/res/a/5/missing":          http.StatusNotFound,
				"/res/a/5/ambiguous/101":    http.StatusUnprocessableEntity,
				"/res/a/5/non-timeline/101": http.StatusUnprocessableEntity,
			} {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
				require.Equal(t, status, w.Code, url+": "+w.Body.String())
			}
			for _, test := range []struct{ url, name string }{
				{"/res/a/5", "101"},
				{"/res/a/5/first", "101"},
				{"/res/a/5/second", "101"},
				{"/res/a/5/direct", "sampled"},
				{"/res/a/5/character", "101"},
			} {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, test.url, nil))
				require.Equal(t, http.StatusOK, w.Code, test.url+": "+w.Body.String())
				var reply unity.Animation
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &reply))
				require.Equal(t, test.name, reply.Name)
			}
			for _, container := range []string{"test", "ambiguous", "non-timeline"} {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/res/a/5/"+container, nil))
				require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())
				require.Contains(t, w.Body.String(), "ambiguous AnimationClip")
			}
		})
	}
	resources = strings.ReplaceAll(resources, "ios-hash", "reloaded-hash")
	require.NoError(t, os.WriteFile(filepath.Join(dbDir, "resources.csv"), []byte(resources), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "reloaded-hash"), fixture, 0o600))
	require.NoError(t, wwwHandler.ReloadMaster())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/res/t/1", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestTexturePNG(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("invalid"), []byte("UnityFS\x00")} {
		_, err := texturePNG(data, "invalid")
		require.Error(t, err)
	}
	data, err := os.ReadFile("internal/unity/testdata/texture.assets")
	require.NoError(t, err)
	encoded, err := texturePNG(data, "test.assets")
	require.NoError(t, err)
	require.NotEmpty(t, encoded)
	data, err = os.ReadFile("internal/unity/testdata/no-texture.assets")
	require.NoError(t, err)
	_, err = texturePNG(data, "test.assets")
	require.EqualError(t, err, "asset contains no Texture2D")
}
