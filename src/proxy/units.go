package proxy

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
)

//go:embed templates/units.html
var unitsHTMLTemplate string

func (h *handler) handleUnits(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, unitsHTMLTemplate)
}

func (h *handler) handleUnitsData(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(h.www.Units())
}
