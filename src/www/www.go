package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/pcommon"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	pb "google.golang.org/protobuf/proto"
)

const protobufContentType = "application/x-protobuf"

type Handler struct {
	router chi.Router
}

func NewHandler() *Handler {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/v1_43_274", func(router chi.Router) {
		router.Post("/account/exist", accountExist)
	})
	router.NotFound(notFound)
	router.MethodNotAllowed(notFound)

	return &Handler{router: router}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func notFound(w http.ResponseWriter, _ *http.Request) {
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
