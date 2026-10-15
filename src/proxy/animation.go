package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"example.com/brave-revival/src/proxy/internal/unity"
	"github.com/go-chi/chi/v5"
)

func (h *handler) handleAnimation(w http.ResponseWriter, r *http.Request) {
	data, hash := h.readResource(w, r)
	if data == nil {
		return
	}
	container := chi.URLParam(r, "container")
	if container == "" {
		container = chi.URLParam(r, "id")
	}
	animation, err := unity.AnimationClip(data, hash, container, chi.URLParam(r, "clip"))
	if err != nil {
		status := http.StatusUnprocessableEntity
		if errors.Is(err, unity.ErrAnimationNotFound) {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	encoded, err := json.Marshal(animation)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	http.ServeContent(w, r, hash+".json", time.Time{}, bytes.NewReader(encoded))
}
