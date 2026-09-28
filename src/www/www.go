package www

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pcommon"
	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/proto"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"
)

const protobufContentType = "application/x-protobuf"

type Handler struct {
	config *config.Config
	router chi.Router

	master    *pmaster.All
	resources config.Resources
	player    *proto.StoredData
}

type handlerKey struct{}

func NewHandler(cfg *config.Config) (*Handler, error) {
	master, err := config.LoadMaster(cfg.DBDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load master: %w", err)
	}

	resources, err := config.LoadResources(cfg.DBDir)
	if err != nil {
		return nil, err
	}
	player, err := loadPlayer(cfg.PlayerPath, master)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	handler := &Handler{
		config:    cfg,
		router:    router,
		master:    master,
		resources: resources,
		player:    player,
	}

	router.Use(middleware.WithValue(handlerKey{}, handler))
	router.Use(middleware.SetHeader("x-enish-app-version-master", fmt.Sprint(master.Version[0].Master)))
	router.Use(middleware.SetHeader("x-enish-app-version-resource", fmt.Sprint(master.Version[0].Resource)))
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/{version:v1_4[34]_274}", func(router chi.Router) {
		router.Get("/etc", etc)
		router.Get("/master/all", masterAll)
		router.Get("/resource/list/{os}", resourceList)
		router.Post("/actionlog/{action}/send", empty("Proto.Empty"))
		router.Post("/fcm/token/add", empty("Proto.Nocontent"))

		router.Post("/account/exist", accountExist)
		router.Post("/account/authorize", accountAuthorize)
		router.Post("/account/certificate", accountCertificate)

		router.Get("/player/list", playerList)
		router.Get("/player/load", playerLoad)
		router.Get("/player/detail/{player_id}", playerDetail)
		router.Post("/player/change/favorite", playerChangeFavorite)

		router.Post("/field/top", fieldTop)
		router.Post("/agito/furniture/set", agitoFurnitureSet)
		router.Post("/agito/item/set", agitoItemSet)

		router.Get("/mission/guild/personal/list", missionGuildPersonalList)
		router.Get("/mission/guild/shared/list", missionGuildSharedList)

		router.Get("/guild/facility/list", empty("Proto.GuildFacilityList"))
	})
	router.Get("/crow/Assets/{os}/{hash}", assets)
	router.Get("/news/top/{os}", news)
	router.NotFound(notFound)
	router.MethodNotAllowed(notFound)

	return handler, nil
}

func loadPlayer(path string, master *pmaster.All) (*proto.StoredData, error) {
	if path == "" {
		return config.GenerateDefaultPlayer(master), nil
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return config.GenerateDefaultPlayer(master), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read player file %q: %w", path, err)
	}
	player := &proto.StoredData{}
	if err := protojson.Unmarshal(data, player); err != nil {
		return nil, fmt.Errorf("failed to parse player file %q: %w", path, err)
	}
	return player, nil
}

func (h *Handler) savePlayer() {
	if h.config.PlayerPath == "" {
		return
	}
	data, err := (protojson.MarshalOptions{Multiline: true}).Marshal(h.player)
	if err != nil {
		slog.Warn("Failed to serialize player", "err", err)
		return
	}
	if err := os.WriteFile(h.config.PlayerPath, append(data, '\n'), 0o644); err != nil {
		slog.Warn("Failed to write player file", "path", h.config.PlayerPath, "err", err)
	}
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

func u64(s string) uint64 {
	result, _ := strconv.ParseUint(s, 10, 64)
	return result
}

func u32(s string) uint32 {
	result, _ := strconv.ParseUint(s, 10, 32)
	return uint32(result)
}
