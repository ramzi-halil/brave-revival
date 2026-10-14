package proxy

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"time"

	"example.com/brave-revival/src/proxy/internal/unity"
	"github.com/go-chi/chi/v5"
)

func (h *handler) handleTexture(w http.ResponseWriter, r *http.Request) {
	data, hash := h.readResource(w, r)
	if data == nil {
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

func (h *handler) readResource(w http.ResponseWriter, r *http.Request) ([]byte, string) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid resource ID", http.StatusBadRequest)
		return nil, ""
	}
	hash, ok := h.www.ResourceHash("ios", uint32(id))
	if !ok {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return nil, ""
	}
	file, err := h.www.OpenResourceByHash(hash)
	if errors.Is(err, fs.ErrNotExist) {
		if androidHash, ok := h.www.ResourceHash("android", uint32(id)); ok {
			hash = androidHash
			file, err = h.www.OpenResourceByHash(hash)
		}
	}
	if err != nil {
		http.Error(w, "Asset not found", http.StatusNotFound)
		return nil, ""
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, ""
	}
	return data, hash
}

func texturePNG(data []byte, name string) ([]byte, error) {
	return unity.TexturePNG(data, name)
}
