package proxy

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"example.com/brave-revival/src/proxy/internal/unity"
	"github.com/go-chi/chi/v5"
)

//go:embed templates/units-model.js
var unitsModelJS string

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
	id, _ := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	names := h.www.ModelMeshNames(uint32(id))
	if len(names) == 0 {
		http.Error(w, "Resource is not an EquipmentCharacter model", http.StatusUnprocessableEntity)
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
