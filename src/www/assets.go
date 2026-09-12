package www

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
)

func assets(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	assetsDir := getHandler(r).config.Assets
	filePath := filepath.Join(assetsDir, hash[:2], hash+".unity3d")
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(w, r, hash, time.Time{}, file)
}
