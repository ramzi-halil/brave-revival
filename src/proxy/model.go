package proxy

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"example.com/brave-revival/src/proxy/internal/unity"
	"github.com/go-chi/chi/v5"
)

//go:embed templates/units-model.js
var unitsModelJS string

//go:embed templates/skeleton.json
var skeletonJSON []byte

func (h *handler) handleSkeleton(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, "skeleton.json", time.Time{}, bytes.NewReader(skeletonJSON))
}

func (h *handler) handleUnitsModelJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, unitsModelJS)
}

func (h *handler) handleModel(w http.ResponseWriter, r *http.Request) {
	data, hash := h.readResource(w, r)
	if data == nil {
		return
	}
	names := r.URL.Query()["name"]
	if len(names) == 0 {
		http.Error(w, "Missing ?name query", http.StatusBadRequest)
		return
	}
	meshes, err := unity.ModelMeshes(data, hash, names...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	encoded, err := json.Marshal(struct {
		Meshes  []unity.Mesh `json:"meshes"`
		Texture string       `json:"texture"`
	}{meshes, "/res/t/" + chi.URLParam(r, "id")})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	http.ServeContent(w, r, hash+".json", time.Time{}, bytes.NewReader(encoded))
}
