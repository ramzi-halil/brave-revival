package proxy

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"time"

	"example.com/brave-revival/src/proxy/internal/unity"
	"github.com/go-chi/chi/v5"
)

func (h *handler) handleTexture(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid resource ID", http.StatusBadRequest)
		return
	}
	hash, ok := h.www.ResourceHash("ios", uint32(id))
	if !ok {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
	file, err := h.www.OpenResourceByHash(hash)
	if err != nil {
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	texture, err := texturePNG(data, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	http.ServeContent(w, r, hash+".png", time.Time{}, bytes.NewReader(texture))
}

func texturePNG(data []byte, name string) ([]byte, error) {
	return unity.TexturePNG(data, name)
}
