package proxy

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"example.com/brave-revival/src/flows"
	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

//go:embed templates/flows.html
var flowsHTMLTemplate string

func (h *handler) handleFlows(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprint(w, flowsHTMLTemplate)
}

func (h *handler) handleFlowEvents(w http.ResponseWriter, r *http.Request) {
	controller := http.NewResponseController(w)
	updates, unsubscribe := h.flows.Subscribe()
	defer unsubscribe()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		data, err := json.Marshal(h.flows.List())
		if err != nil {
			return
		}
		controller.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return
		}
		if err := controller.Flush(); err != nil {
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-updates:
		case <-ticker.C:
		}
	}
}

func (h *handler) handleFlow(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid flow ID", http.StatusBadRequest)
		return
	}
	record, ok := h.flows.Get(id)
	if !ok {
		http.Error(w, "Flow is no longer available", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		flows.Record
		RequestText  string `json:"requestText"`
		ResponseText string `json:"responseText"`
	}{record, bodyText(record.RequestBody), responseText(record)})
}

func bodyText(body []byte) string {
	if utf8.Valid(body) {
		return string(body)
	}
	return "Base64:\n" + base64.StdEncoding.EncodeToString(body)
}

func responseText(record flows.Record) string {
	if record.ProtoType != "" {
		messageType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(record.ProtoType))
		if err == nil {
			message := messageType.New().Interface()
			if err := pb.Unmarshal(record.ResponseBody, message); err == nil {
				if data, err := (protojson.MarshalOptions{Multiline: true, UseProtoNames: true}).Marshal(message); err == nil {
					return string(data)
				}
			}
		}
	}
	return bodyText(record.ResponseBody)
}
