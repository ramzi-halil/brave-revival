package www

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func assets(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	file, err := getHandler(r).OpenResourceByHash(hash)
	if err != nil {
		slog.Warn("Asset not found", "hash", hash, "error", err)
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}

	defer file.Close()
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeContent(w, r, hash, time.Time{}, file)
}
