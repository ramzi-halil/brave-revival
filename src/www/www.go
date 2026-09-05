package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/pcommon"
	"example.com/brave-revival/src/proto/proto"
	"github.com/go-chi/chi/v5"
	pb "google.golang.org/protobuf/proto"
)

const protobufContentType = "application/x-protobuf"

type Handler struct {
	router chi.Router
}

func NewHandler() *Handler {
	router := chi.NewRouter()
	h := &Handler{router: router}

	router.Route("/v1_43_274", func(router chi.Router) {
		router.Post("/account/exist", h.accountExist)
	})
	router.NotFound(h.notFound)
	router.MethodNotAllowed(h.notFound)

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) accountExist(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusOK, &proto.PlayerExist{
		PlayerSummary: &proto.PlayerSummary{
			PlayerId: 100,
			Nickname: "Dummy Player",
			JobId:    1,
			JobLevel: 999,
			Power:    99_999_999,
		},
		WorldDescription: "Dummy World",
	})
}

func (h *Handler) notFound(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusNotFound, &pcommon.Error{
		Code: 404,
		Msg:  "Not Found",
	})
}

func writeProto(w http.ResponseWriter, status int, message pb.Message) {
	w.Header().Set("Content-Type", protobufContentType)
	w.Header().Set("proto-type", string(message.ProtoReflect().Descriptor().FullName()))
	payload, err := pb.Marshal(message)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	_, _ = w.Write(payload)
}
