package www

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestReloadPlayer(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player.json")
	oldPlayer := &proto.StoredData{Generation: 1}
	h := &Handler{config: &config.Config{PlayerPath: path}, master: &pmaster.All{}, player: oldPlayer}
	data, err := protojson.Marshal(&proto.StoredData{Generation: 7, Player: &puser.Player{Nickname: "reloaded"}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))

	require.NoError(t, h.ReloadPlayer())
	require.Equal(t, uint64(7), h.player.Generation)
	require.Equal(t, "reloaded", h.player.Player.Nickname)

	reloaded := h.player
	require.NoError(t, os.WriteFile(path, []byte("invalid"), 0o600))
	require.Error(t, h.ReloadPlayer())
	require.Same(t, reloaded, h.player)
}

func TestReloadMaster(t *testing.T) {
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
			content += "ios,1,2,3\n"
		}
		require.NoError(t, os.WriteFile(filepath.Join(dbDir, string(field.Name())+".csv"), []byte(content), 0o600))
	}
	resourcesPath := filepath.Join(dbDir, "resources.csv")
	require.NoError(t, os.WriteFile(resourcesPath, []byte("id,ios,android\n1,new-hash,other-hash\n"), 0o600))

	oldMaster := &pmaster.All{Version: []*pmaster.Version{{Master: 9}}}
	h := &Handler{config: &config.Config{DBDir: dbDir}, master: oldMaster}
	require.NoError(t, h.ReloadMaster())
	require.Equal(t, uint32(3), h.master.Version[0].Master)
	require.Equal(t, "new-hash", h.resources["ios"].Resource[1].Hash)

	w := httptest.NewRecorder()
	h.versionHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {})).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	require.Equal(t, "3", w.Header().Get("x-enish-app-version-master"))
	require.Equal(t, "2", w.Header().Get("x-enish-app-version-resource"))

	reloadedMaster := h.master
	reloadedResources := h.resources["ios"]
	require.NoError(t, os.WriteFile(resourcesPath, []byte("invalid\n"), 0o600))
	require.Error(t, h.ReloadMaster())
	require.Same(t, reloadedMaster, h.master)
	require.Same(t, reloadedResources, h.resources["ios"])
}
