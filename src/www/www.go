package www

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

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
	resources config.Resources
	player    *proto.PlayerDetail
}

type handlerKey struct{}

func NewHandler(cfg *config.Config, player *proto.PlayerDetail) (*Handler, error) {
	// TODO: Don't use pre-compiled master & resources.
	masterBytes, err := os.ReadFile("./patched.pb")
	if err != nil {
		return nil, err
	}
	var master pmaster.All
	pb.Unmarshal(masterBytes, &master)

	resourcesFile, err := os.Open(filepath.Join(cfg.DBDir, "res.csv"))
	if err != nil {
		return nil, err
	}
	defer resourcesFile.Close()
	resources, err := config.LoadResources(resourcesFile)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	handler := &Handler{
		config:    cfg,
		router:    router,
		master:    &master,
		resources: resources,
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
		router.Post("/actionlog/{action}/send", empty("Proto.Empty"))

		router.Post("/account/exist", accountExist)
		router.Post("/account/authorize", accountAuthorize)
		router.Post("/account/certificate", accountCertificate)

		router.Get("/player/list", playerList)
		router.Get("/player/load", playerLoad)

		router.Get("/mission/guild/personal/list", missionGuildPersonalList)
		router.Get("/mission/guild/shared/list", missionGuildSharedList)
	})
	router.Get("/crow/Assets/{os}/{hash}", assets)
	router.NotFound(notFound)
	router.MethodNotAllowed(notFound)

	return handler, nil
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

func empty(protoType string) func(w http.ResponseWriter, _ *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", protobufContentType)
		w.Header().Set("proto-type", protoType)
		w.WriteHeader(http.StatusOK)
	}
}
