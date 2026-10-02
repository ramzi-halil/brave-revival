package proxy

import (
	"crypto/tls"
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
	"example.com/brave-revival/src/www"
	"github.com/kvarenzn/ssm/uni"
	"github.com/stretchr/testify/require"
)

func TestTextureViewer(t *testing.T) {
	fixture, err := os.ReadFile("testdata/texture.unity3d")
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
		}
		require.NoError(t, os.WriteFile(filepath.Join(dbDir, string(field.Name())+".csv"), []byte(content), 0o600))
	}
	resources := "id,ios,android,ios_size,android_size\n1,ios-hash,android-hash,440,0\n2,missing-hash,android-hash,0,0\n3,broken-hash,android-hash,0,0\n"
	require.NoError(t, os.WriteFile(filepath.Join(dbDir, "resources.csv"), []byte(resources), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(assetsDir, "broken-hash"), []byte("UnityFS\x00"), 0o600))
	wwwHandler, err := www.NewHandler(&config.Config{DBDir: dbDir, AssetsDir: assetsDir})
	require.NoError(t, err)
	h := newHandler(&config.Config{}, tls.Certificate{}, wwwHandler)

	for _, path := range []string{"io/ios-hash.unity3d", "ios-hash"} {
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
	data, err := os.ReadFile("testdata/texture.unity3d")
	require.NoError(t, err)
	reader, err := uni.NewFileReader(data, "test")
	require.NoError(t, err)
	bundle, err := uni.NewBundleFile(reader)
	require.NoError(t, err)
	_, err = texturePNG(bundle.Files[0].Stream, "test.assets")
	require.Error(t, err) // The streamed pixels require the bundle's .resS entry.
	data, err = os.ReadFile("testdata/texture.assets")
	require.NoError(t, err)
	encoded, err := texturePNG(data, "test.assets")
	require.NoError(t, err)
	require.NotEmpty(t, encoded)
	data, err = os.ReadFile("testdata/no-texture.assets")
	require.NoError(t, err)
	_, err = texturePNG(data, "test.assets")
	require.EqualError(t, err, "asset contains no Texture2D")
}

func TestDecodeASTCTexture(t *testing.T) {
	block := []byte{0xfc, 0xfd, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0xff, 0xff}
	for _, format := range []uni.TextureFormat{uni.ASTC_RGB_5x5, uni.ASTC_RGBA_5x5} {
		texture := &uni.Texture2D{
			Width: 5, Height: 5, Format: format,
			ImageData: uni.NewResourceReader(uni.NewBinaryReaderFromBytes(block, true), 0, int64(len(block))),
		}
		img, err := decodeTexture(texture)
		require.NoError(t, err)
		require.Equal(t, color.NRGBA{R: 255, A: 255}, color.NRGBAModel.Convert(img.At(0, 0)))
		texture.ImageData.Size = 1
		_, err = decodeTexture(texture)
		require.Error(t, err)
	}
}
