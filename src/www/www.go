package www

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/flows"
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
	flows  *flows.Recorder

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
		flows:     &flows.Recorder{},
		master:    master,
		resources: resources,
		player:    player,
	}

	router.Use(middleware.WithValue(handlerKey{}, handler))
	router.Use(handler.versionHeaders)
	router.Use(middleware.Recoverer)

	router.Route("/{version:v1_4[34]_274}", func(router chi.Router) {
		router.Use(handler.flows.Middleware)
		// Recover inside the recorder so panic responses are captured too.
		router.Use(middleware.Recoverer)
		router.Get("/etc", etc)
		router.Get("/master/all", masterAll)
		router.Get("/resource/list/{os}", resourceList)
		router.Post("/actionlog/{action}/send", empty("Proto.Empty"))
		router.Post("/fcm/token/add", empty("Proto.Nocontent"))
		router.Post("/season_pass/top", empty("Proto.SeasonPassTop"))

		router.Post("/account/exist", accountExist)
		router.Post("/account/authorize", accountAuthorize)
		router.Post("/account/certificate", accountCertificate)
		router.Post("/account/inherit/password", accountInheritPassword)
		router.Post("/account/reset", empty("Proto.Nocontent"))
		router.Post("/account/inherit", empty("Proto.Nocontent"))

		router.Get("/player/list", playerList)
		router.Get("/player/load", playerLoad)
		router.Get("/player/detail/{player_id}", playerDetail)
		router.Post("/setting/update", empty("Proto.Nocontent"))

		router.Post("/player/change/favorite", playerChangeFavorite)
		router.Post("/player/delete", empty("Proto.Nocontent"))
		router.Post("/player/change/nickname", playerChangeNickname)
		router.Post("/player/change/comment", playerChangeComment)
		router.Post("/player/summary/list", playerSummaryList)
		router.Post("/title/set", titleSet)

		router.Get("/friend/list", empty("Proto.FriendList"))
		router.Get("/friend/approval/list", empty("Proto.FriendApprovalList"))
		router.Get("/friend/request/list", empty("Proto.FriendRequestList"))
		router.Get("/player/recommend/list", empty("Proto.PlayerRecommendList"))
		router.Post("/player/search/list", empty("Proto.PlayerSearchList"))
		router.Get("/block/list", empty("Proto.BlockList"))

		router.Get("/mercenary/history", empty("Proto.MercenaryHistory"))
		router.Post("/mercenary/register", empty("Proto.Nocontent"))
		router.Post("/mercenary/cancel", empty("Proto.Nocontent"))
		router.Get("/mercenary/list", empty("Proto.MercenaryList"))

		router.Post("/field/top", fieldTop)
		router.Post("/agito/furniture/set", agitoFurnitureSet)
		router.Post("/agito/item/set", agitoItemSet)
		router.Post("/agito/apitem/set", empty("Proto.Nocontent"))
		router.Get("/agito/list/good", empty("Proto.AgitoListGoodHistoryList"))
		router.Post("/agito/player/recommend", agitoPlayerRecommend)
		router.Post("/agito/player/good", empty("Proto.Nocontent"))
		router.Post("/agito/item/receive", agitoItemReceive)

		router.Get("/mission/guild/personal/list", missionGuildPersonalList)
		router.Get("/mission/guild/shared/list", missionGuildSharedList)
		router.Get("/loginbonus/list", empty("Proto.UserLoginBonus"))

		router.Get("/guild/player/invite/request/list", empty("Proto.GuildInviteRequestList"))
		router.Get("/guild/player/join/request/list", empty("Proto.GuildJoinRequestList"))
		router.Get("/guild/player/info", empty("Proto.GuildPlayerInfo"))
		router.Get("/guild/recommend/list", empty("Proto.GuildRecommendList"))
		router.Post("/guild/search/list", empty("Proto.GuildSearchList"))
		router.Get("/guild/penalty", guildPenalty)
		router.Post("/guild/create", empty("Proto.Nocontent"))
		router.Post("/guild/summary/list", guildSummaryList)

		router.Get("/guild/facility/list", empty("Proto.GuildFacilityList"))
		router.Post("/guild/top", guildDetail)
		router.Post("/guild/detail", guildDetail)
		router.Post("/guild/detail/other", guildDetail)
		router.Get("/guild/member/list/{gid}", empty("Proto.GuildMemberList"))
		router.Get("/guild/join/request/list", empty("Proto.GuildJoinRequestList"))
		router.Get("/guild/invite/request/list", empty("Proto.GuildInviteRequestList"))
		router.Get("/guild/login/summary", guildLoginSummary)
		router.Get("/guild/login/member/list", empty("Proto.GuildLoginMemberList"))
		router.Post("/guild/donate", empty("Proto.Nocontent"))
		router.Post("/guild/donate/all", empty("Proto.Nocontent"))
		router.Get("/guild/history/list", empty("Proto.GuildHistoryList"))
		router.Get("/guild/warehouse/list", empty("Proto.GuildWarehouseList"))
		router.Get("/guild/warehouse/history/list", empty("Proto.GuildWarehouseHistoryList"))
		router.Post("/guild/update/symbol", guildUpdateSymbol)
		router.Post("/guild/member/leave", empty("Proto.Nocontent"))
		router.Post("/guild/update/setting", empty("Proto.Nocontent"))
		router.Post("/guild/update/description", empty("Proto.Nocontent"))
		router.Post("/guild/change/name", guildChangeName)
		router.Post("/guild/update/boardtitle", empty("Proto.Nocontent"))
		router.Post("/guild/update/boardmessage", empty("Proto.Nocontent"))
		router.Post("/guild/update/boardshow", empty("Proto.Nocontent"))
		router.Post("/guild/dungeon/top", guildDungeonTop)
		router.Get("/mission/guild/weekly/list", missionGuildWeeklyList)
		router.Get("/holybeast/top", empty("Proto.HolyBeastInfo"))

		router.Get("/gvg/bid/top", empty("Proto.GvgBidTop"))
		router.Get("/gvg/bid/top/field/list", gvgBidTopFieldList)
		router.Get("/gvg/history/list", gvgHistoryList)
		// router.Get("/gvg/practice/top", gvgPracticeTop)
		router.Get("/gvg/practice/history/list", empty("Proto.GvgHistoryList"))

		router.Post("/vip/receive/daily/reward", empty("Proto.Nocontent"))
		router.Post("/shop/buy", shopBuy)
		router.Post("/shop/top", empty("Proto.ShopTopResponse"))

		router.Post("/gacha/purchase", gachaPurchase)
		router.Post("/event/roulette/play", empty("Proto.RoulettePlay"))
		router.Post("/item/exchange", empty("Proto.Nocontent"))
		router.Post("/item/use/box/choice", empty("Proto.UseItemBoxReceiveList"))
		router.Post("/item/use/box/random", empty("Proto.UseItemBoxReceiveList"))
		router.Post("/item/use", empty("Proto.Nocontent"))

		router.Get("/ranking/list", empty("Proto.RankingList"))
		router.Post("/present/list", empty("Proto.PresentBoxList"))

		router.Post("/elixir/manufacture", empty("Proto.Nocontent"))
		router.Post("/anima/put", empty("Proto.Nocontent"))
		router.Post("/achievement/daily", empty("Proto.Nocontent"))

		router.Get("/chat/messages", empty("Proto.ChatMessageListResponse"))
		router.Get("/chat/directs", empty("Proto.ChatFriendListResponse"))
		router.Get("/chat/friends", empty("Proto.ChatFriendListResponse"))
		router.Get("/chat/groups", empty("Proto.ChatGroupWithInviteResponse"))
		router.Post("/chat/group/create", empty("Proto.ChatGroupCreateResultResponse"))

		router.Post("/equipment/forge", equipmentForge)
		router.Post("/equipment/bulkforge", equipmentForge)
		router.Post("/equipment/enhance", equipmentEnhance)
		router.Post("/equipment/inherit/enhancement", equipmentInheritEnhance)
		router.Post("/equipment/option_skill/redraw", empty("Proto.Nocontent"))
		router.Post("/equipment/option_skill/lock", equipmentOptionSkillLock)
		router.Post("/equipment/option_skill/unlock", equipmentOptionSkillUnlock)
		router.Post("/equipment/limitbreak", equipmentLimitbreak)
		router.Post("/equipment/evolution/material", equipmentEvolutionMaterial)
		router.Post("/equipment/weapon_skill/enhance/material", equipmentWeaponSkillEnhanceMaterial)
		router.Post("/equipment/rune/detach", equipmentRuneDetach)
		router.Post("/equipment/rune/detach/all", equipmentRuneDetachAll)
		router.Post("/equipment/rune/attach", equipmentRuneAttach)
		router.Post("/equipment/record", empty("Proto.EquipmentRecordList"))
		router.Post("/equipment/sell", empty("Proto.Nocontent"))
		router.Post("/equipment/protect/lock", equipmentProtectLock)
		router.Post("/equipment/protect/unlock", equipmentProtectUnlock)
		router.Post("/equipment/awakening", equipmentAwakening)
		router.Post("/equipment/awakeningreset", equipmentAwakeningreset)

		router.Post("/player/change/job", playerChangeJob)
		router.Post("/job/skill/learn", jobSkillLearn)
		router.Post("/job/skill/enhance", jobSkillEnhance)
		router.Post("/job/skill/reset", jobSkillReset)
		router.Post("/job/deck/change", jobDeckChange)
		router.Post("/job/deck/set", jobDeckSet)
		router.Post("/job/group/label/change", jobGroupLabelChange)
		router.Post("/job/deck/label/change", jobDeckLabelChange)
		router.Post("/job/deck/equipment/remove/all", jobDeckEquipmentRemoveAll)

		router.Post("/abyss/fever/charge", empty("Proto.Nocontent"))
		router.Get("/tower/top", empty("Proto.TowerTopResult"))
		router.Post("/battle/tower/sweep", battleTowerSweep)
		router.Get("/battle/raid/history", empty("Proto.RaidHistoryResponse"))

		router.Get("/arena/season", arenaSeason)
		router.Get("/arena/ranking", empty("Proto.ArenaRankingResponse"))
		router.Get("/arena/history", empty("Proto.ArenaHistoryResponse"))
		router.Post("/arena/jobdeck/set", arenaJobdeckSet)
		// TODO: actually return a valid opponent when we can play battle.
		router.Get("/arena/opponent", empty("Proto.ArenaOpponentResponse"))

		router.Post("/rune/sell", empty("Proto.Nocontent"))
		router.Post("/rune/rarity/up", empty("Proto.Nocontent"))
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

func (h *Handler) Flows() *flows.Recorder {
	return h.flows
}

func (h *Handler) ResourceHash(platform string, id uint32) (string, bool) {
	h.masterLock.Lock()
	defer h.masterLock.Unlock()
	resources := h.resources[platform]
	if resources == nil || resources.Resource[id] == nil {
		return "", false
	}
	hash := resources.Resource[id].Hash
	return hash, hash != ""
}

func (h *Handler) OpenResourceByHash(hash string) (*os.File, error) {
	if len(hash) < 2 {
		return nil, os.ErrNotExist
	}
	for _, path := range []string{hash[:2] + "/" + hash + ".unity3d", hash} {
		file, err := h.assets.Open(path)
		if err == nil {
			return file, nil
		}
	}
	return nil, os.ErrNotExist
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

func readProto(r *http.Request, message pb.Message) bool {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("failed to read request body", "error", err)
		return false
	}
	if err := pb.Unmarshal(body, message); err != nil {
		slog.Error("failed to unmarshal protobuf", "error", err)
		return false
	}
	return true
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
