package www

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"

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
	assets *os.Root

	masterLock sync.Mutex
	master     *pmaster.All
	resources  config.Resources

	playerLock sync.Mutex
	player     *proto.StoredData
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

	assetsDir, err := os.OpenRoot(cfg.AssetsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to open assets dir: %w", err)
	}

	router := chi.NewRouter()
	handler := &Handler{
		config:    cfg,
		router:    router,
		assets:    assetsDir,
		master:    master,
		resources: resources,
		player:    player,
	}

	router.Use(middleware.WithValue(handlerKey{}, handler))
	router.Use(handler.versionHeaders)
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
	router.Get("/crow/Assets/{os}/{hash:[0-9a-f]{32}}", assets)
	router.Get("/news/top/{os}", news)
	router.NotFound(notFound)
	router.MethodNotAllowed(notFound)

	return handler, nil
}

func (h *Handler) versionHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.masterLock.Lock()
		version := h.master.Version[0]
		w.Header().Set("x-enish-app-version-master", fmt.Sprint(version.Master))
		w.Header().Set("x-enish-app-version-resource", fmt.Sprint(version.Resource))
		h.masterLock.Unlock()
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) ReloadPlayer() error {
	h.masterLock.Lock()
	master := h.master
	h.masterLock.Unlock()

	player, err := loadPlayer(h.config.PlayerPath, master)
	if err != nil {
		return err
	}
	h.playerLock.Lock()
	h.player = player
	h.playerLock.Unlock()
	return nil
}

func (h *Handler) ReloadMaster() error {
	master, err := config.LoadMaster(h.config.DBDir)
	if err != nil {
		return fmt.Errorf("failed to load master: %w", err)
	}
	if len(master.Version) == 0 {
		return fmt.Errorf("loaded master has no version")
	}
	resources, err := config.LoadResources(h.config.DBDir)
	if err != nil {
		return err
	}
	h.masterLock.Lock()
	h.master = master
	h.resources = resources
	h.masterLock.Unlock()
	return nil
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

func (h *Handler) readPlayer[R any](action func(player *proto.StoredData) R) R {
	h.playerLock.Lock()
	defer h.playerLock.Unlock()
	return action(h.player)
}

func (h *Handler) writePlayer[R any](action func(player *proto.StoredData) R) R {
	locked := true
	h.playerLock.Lock()
	defer func() {
		if locked {
			h.playerLock.Unlock()
		}
	}()

	h.player.Generation++
	result := action(h.player)

	if h.config.PlayerPath != "" {
		data, err := (protojson.MarshalOptions{Multiline: true}).Marshal(h.player)
		if err == nil {
			h.playerLock.Unlock()
			locked = false
			if err := os.WriteFile(h.config.PlayerPath, data, 0o644); err != nil {
				slog.Warn("Failed to write player file", "path", h.config.PlayerPath, "err", err)
			}
		} else {
			slog.Warn("Failed to serialize player", "err", err)
		}
	}
	return result
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
	writeProtoStreamed(w, status, func(stream func(pb.Message) error) error {
		return stream(message)
	})
}

func writeProtoStreamed(w http.ResponseWriter, status int, action func(func(pb.Message) error) error) {
	var protoType string
	var payload []byte

	if err := action(func(message pb.Message) (err error) {
		protoType = string(message.ProtoReflect().Descriptor().FullName())
		payload, err = pb.Marshal(message)
		return
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", protobufContentType)
	w.Header().Set("proto-type", protoType)
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
