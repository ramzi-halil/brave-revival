package proxy

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/www"
	"github.com/stretchr/testify/require"
)

func TestAgitoEditorRoutes(t *testing.T) {
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
		switch field.Name() {
		case "version":
			content += "ios,1.43.274,6749,12810\n"
		case "agito_item_area":
			content += "101,1,0,0,0\n"
		case "agito_item":
			content += "10,100,0,0,4,0,0,0,999\n"
		case "agito_visitor_lineup":
			content += "11,200,1,1,30,1,0,0\n"
		}
		require.NoError(t, os.WriteFile(filepath.Join(dbDir, string(field.Name())+".csv"), []byte(content), 0o600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(dbDir, "resources.csv"), []byte("id,ios,android,ios_size,android_size\n"), 0o600))
	playerPath := filepath.Join(t.TempDir(), "player.json")
	wwwHandler, err := www.NewHandler(&config.Config{DBDir: dbDir, AssetsDir: t.TempDir(), PlayerPath: playerPath})
	require.NoError(t, err)
	h := newHandler(&config.Config{}, tls.Certificate{}, wwwHandler)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/agito", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), "Agito editor")
	require.Contains(t, w.Body.String(), "Other choices")
	require.Contains(t, w.Body.String(), "/res/t/")

	for _, body := range []string{"invalid", `null`, `{}`, `[{"areaId":101,"unexpected":true}]`, `[] []`, `[{"areaId":999}]`} {
		w = httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/forms/agito", strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, w.Code, body)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPut, "/forms/agito", strings.NewReader(`[{"areaId":101,"itemId":10,"lineupIds":[11,0,0]}]`)))
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())
	require.NoError(t, wwwHandler.ReloadPlayer())
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/forms/agito", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	var data www.AgitoEditorData
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &data))
	require.Equal(t, uint32(10), data.Areas[0].ItemID)
	require.Equal(t, []uint32{11, 0, 0}, data.Areas[0].LineupIDs)
}
