package www

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmaster"
	"github.com/stretchr/testify/require"
)

func TestPlayerPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player.bak")
	master := &pmaster.All{}
	player, err := loadPlayer(path, master)
	require.NoError(t, err)
	require.Equal(t, uint64(1), player.Generation)

	handler := &Handler{config: &config.Config{PlayerPath: path}, player: player}
	testMutation := func(endpoint string, form url.Values) {
		t.Helper()
		r := httptest.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r = r.WithContext(context.WithValue(r.Context(), handlerKey{}, handler))
		w := httptest.NewRecorder()
		switch endpoint {
		case "/player/change/favorite":
			playerChangeFavorite(w, r)
		case "/agito/furniture/set":
			agitoFurnitureSet(w, r)
		}
		require.Equal(t, http.StatusOK, w.Code)
	}

	testMutation("/player/change/favorite", url.Values{"equipment_id_1": {"12345"}})
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.True(t, json.Valid(data))
	require.Contains(t, string(data), "\n  \"")
	saved, err := loadPlayer(path, master)
	require.NoError(t, err)
	require.Equal(t, uint64(2), saved.Generation)
	require.Equal(t, uint64(12345), saved.Player.FavoriteEquipmentId_1)

	ids := make([]string, 15)
	for i := range ids {
		ids[i] = "42"
	}
	testMutation("/agito/furniture/set", url.Values{"item_ids": ids})

	reloaded, err := loadPlayer(path, master)
	require.NoError(t, err)
	require.Equal(t, uint64(3), reloaded.Generation)
	require.Equal(t, uint64(12345), reloaded.Player.FavoriteEquipmentId_1)
	require.Equal(t, uint32(42), reloaded.AgitoFurnitureSetting.List[1].WallPaperItemId)
}

func TestPlayerMutationIgnoresSaveFailure(t *testing.T) {
	master := &pmaster.All{}
	player, err := loadPlayer("", master)
	require.NoError(t, err)
	handler := &Handler{config: &config.Config{PlayerPath: t.TempDir()}, player: player}
	r := httptest.NewRequest(http.MethodPost, "/player/change/favorite", strings.NewReader("equipment_id_1=12345"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r = r.WithContext(context.WithValue(r.Context(), handlerKey{}, handler))
	w := httptest.NewRecorder()
	playerChangeFavorite(w, r)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, uint64(12345), player.Player.FavoriteEquipmentId_1)
}

func TestLoadPlayerRejectsInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "player.bak")
	require.NoError(t, os.WriteFile(path, []byte{0xff}, 0o600))
	_, err := loadPlayer(path, &pmaster.All{})
	require.ErrorContains(t, err, "failed to parse player file")
}
