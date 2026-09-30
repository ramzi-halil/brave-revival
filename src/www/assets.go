package www

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func assets(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	assetsDir := getHandler(r).assets
	for _, path := range []string{hash[:2] + "/" + hash + ".unity3d", hash} {
		file, err := assetsDir.Open(path)
		if err == nil {
			defer file.Close()
			w.Header().Set("Content-Type", "application/octet-stream")
			http.ServeContent(w, r, hash, time.Time{}, file)
			return
		}
	}

	http.Error(w, "Asset not found", http.StatusNotFound)
}
