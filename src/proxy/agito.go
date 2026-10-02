package proxy

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"example.com/brave-revival/src/www"
)

//go:embed templates/agito.html
var agitoHTMLTemplate string

func (h *handler) handleAgito(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, agitoHTMLTemplate)
}

func (h *handler) handleAgitoData(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(h.www.AgitoEditor())
}

func (h *handler) handleAgitoSave(w http.ResponseWriter, r *http.Request) {
	var selections []www.AgitoSelection
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&selections); err != nil {
		http.Error(w, "Invalid Agito selections: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF || selections == nil {
		http.Error(w, "Expected a single JSON array", http.StatusBadRequest)
		return
	}
	if err := h.www.SaveAgitoEditor(selections); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, www.ErrInvalidAgitoSelection) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
