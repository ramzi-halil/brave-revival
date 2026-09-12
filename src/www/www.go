package www

import (
	"fmt"
	"net/http"
	"os"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pcommon"
	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/proto"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	pb "google.golang.org/protobuf/proto"
)

const protobufContentType = "application/x-protobuf"

type Handler struct {
	config *config.Config
	router chi.Router

	master    *pmaster.All
	resources *pmaster.Resources
	player    *proto.PlayerDetail
}

type handlerKey struct{}

func NewHandler(config *config.Config, player *proto.PlayerDetail) *Handler {
	// TODO: Don't use pre-compiled master & resources.
	masterBytes, err := os.ReadFile("./patched.pb")
	if err != nil {
		panic(err)
	}
	resourcesBytes, err := os.ReadFile("./v1_43_274-res.pb")
	if err != nil {
		panic(err)
	}

	var master pmaster.All
	pb.Unmarshal(masterBytes, &master)
	var resources pmaster.Resources
	pb.Unmarshal(resourcesBytes, &resources)

	router := chi.NewRouter()
	handler := &Handler{
		config:    config,
		router:    router,
		master:    &master,
		resources: &resources,
		player:    player,
	}

	router.Use(middleware.WithValue(handlerKey{}, handler))
	router.Use(middleware.SetHeader("x-enish-app-version-master", fmt.Sprint(master.Version[0].Master)))
	router.Use(middleware.SetHeader("x-enish-app-version-resource", fmt.Sprint(master.Version[0].Resource)))
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/v1_43_274", func(router chi.Router) {
		router.Get("/etc", etc)
		router.Get("/master/all", masterAll)
		router.Get("/resource/list/{os}", resourceList)
		router.Post("/actionlog/{action}/send", actionlog)

		router.Post("/account/exist", accountExist)
		router.Post("/account/authorize", accountAuthorize)
		router.Post("/account/certificate", accountCertificate)

		router.Get("/player/list", playerList)
	})
	router.Get("/crow/Assets/{os}/{hash}", assets)
	router.NotFound(notFound)
	router.MethodNotAllowed(notFound)

	return handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func getHandler(r *http.Request) *Handler {
	return r.Context().Value(handlerKey{}).(*Handler)
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
