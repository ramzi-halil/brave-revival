package www

import (
	"net/http"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmisc"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

func generateGuild(player *proto.StoredData) *pmisc.Guild {
	return &pmisc.Guild{
		Id:                      player.GuildInfo.GuildId,
		Name:                    player.GuildInfo.GuildName,
		Exp:                     99_999_999,
		Lupi:                    999_999_999,
		Wood:                    999_999,
		Stone:                   999_999,
		Iron:                    999_999,
		Crystal:                 999_999,
		FacilityItemCount:       999,
		Star:                    999_999_999,
		Moon:                    999_999_999,
		Description:             "hello",
		JoinType:                1,
		PlayStyle:               2,
		RecruitmentTarget:       1,
		DungeonType:             1,
		DungeonResetType:        2,
		Symbol:                  player.GuildInfo.GuildSymbol,
		SymbolFrame:             player.GuildInfo.GuildSymbolFrame,
		SymbolFrameColor:        player.GuildInfo.GuildSymbolFrameColor,
		RewardLv:                50,
		UpdateRewardLvDate:      "0",
		WarehouseGiftReceivedAt: "0",
		NameChangedAt:           "0",
		GvgPracticeLimitAt:      "0",
	}
}

func guildDetail(w http.ResponseWriter, r *http.Request) {
	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.GuildDetail {
		guildBoards := make([]*pmisc.GuildBoard, 0, 6)
		for i := range uint32(6) {
			isTop := uint32(0)
			if i == 0 {
				isTop = 1
			}
			guildBoards = append(guildBoards, &pmisc.GuildBoard{
				GuildId:    player.GuildInfo.GuildId,
				Seq:        i + 1,
				IsGuildTop: isTop,
				IsGveTop:   isTop,
			})
		}

		return &proto.GuildDetail{
			Guild:                generateGuild(player),
			Members:              []*proto.GuildMember{{PlayerSummary: config.GeneratePlayerSummary(player)}},
			Lv:                   50,
			IsRecommend:          1,
			MaxMemberCount:       40,
			EnableUpdateSymbolAt: "0",
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				GuildInfo:  player.GuildInfo,
			},
			IsLogined: 1,
			Role:      1,
			Donation: &pmisc.GuildDonation{
				GuildId:  player.GuildInfo.GuildId,
				PlayerId: player.Player.Id,
			},
			GuildBoards: guildBoards,
			GvgBattleField: &pmisc.GvgBattleField{
				StartBattleDate: "0",
			},
			GuildDungeonRaidEndAt: "0",
			HolybeastRaidEndAt:    "0",
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func guildLoginSummary(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusOK, &proto.GuildLoginSummary{
		TodayLoginMemberCount:     1,
		YesterdayLoginMemberCount: 1,
		TodayMaxMemberCount:       1,
		YesterdayMaxMemberCount:   1,
		GuildLv:                   50,
		RewardGuildLv:             50,
	})
}

func guildUpdateSymbol(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	symbol := u32(r.PostForm.Get("symbol"))
	symbolFrame := u32(r.PostForm.Get("symbol_frame"))
	symbolFrameColor := u32(r.PostForm.Get("symbol_frame_color"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		player.GuildInfo.GuildSymbol = symbol
		player.GuildInfo.GuildSymbolFrame = symbolFrame
		player.GuildInfo.GuildSymbolFrameColor = symbolFrameColor
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				GuildInfo:  player.GuildInfo,
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func guildPenalty(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusOK, &proto.GuildPenalty{
		LastLeavingAt: "0",
	})
}

func missionGuildWeeklyList(w http.ResponseWriter, r *http.Request) {
	handler := getHandler(r)
	handler.masterLock.Lock()
	anyMissionGroup := handler.master.GuildWeeklyMissionGroup[0].Id
	handler.masterLock.Unlock()

	mission := &proto.GuildWeeklyMission{
		MissionGroupId: anyMissionGroup,
		DayMonday:      "0",
	}
	writeProto(w, http.StatusOK, &proto.GuildWeeklyMissionResult{
		List:   []*proto.GuildWeeklyMission{mission, mission, mission},
		Reward: &puser.GuildWeeklyMissionRewardList{},
	})
}

func guildChangeName(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	name := r.PostForm.Get("name")
	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		player.GuildInfo.GuildName = name
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				GuildInfo:  player.GuildInfo,
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func guildDungeonTop(w http.ResponseWriter, r *http.Request) {
	type stageKey struct {
		areaID     uint32
		difficulty uint32
	}

	dungeons := make(map[uint32]*proto.GuildDungeon)
	stageLists := make(map[stageKey]map[uint32]uint32)

	handler := getHandler(r)
	handler.masterLock.Lock()
	for _, dungeon := range handler.master.GuildDungeon {
		key := stageKey{areaID: dungeon.AreaId, difficulty: dungeon.Difficulty}
		stageLists[key] = make(map[uint32]uint32)
	}
	for _, stage := range handler.master.Stage {
		key := stageKey{areaID: stage.AreaId, difficulty: stage.Difficulty}
		if s, ok := stageLists[key]; ok {
			s[stage.Id] = 0
		}
	}
	for _, dungeon := range handler.master.GuildDungeon {
		key := stageKey{areaID: dungeon.AreaId, difficulty: dungeon.Difficulty}
		dungeons[dungeon.Id] = &proto.GuildDungeon{
			GuildDungeon: &pmisc.GuildDungeon{
				GuildDungeonId: dungeon.Id,
				IsOnceCleared:  1,
				LastOpenAt:     "0",
				LastResetAt:    "0",
				LastClearAt:    "0",
				LastUpdateAt:   "0",
				RaidEndAt:      "0",
			},
			StageList: stageLists[key],
		}
	}
	handler.masterLock.Unlock()

	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.GuildDungeonTop {
		for _, dungeon := range dungeons {
			dungeon.GuildDungeon.GuildId = player.GuildInfo.GuildId
		}
		return &proto.GuildDungeonTop{
			Guild:            generateGuild(player),
			GuildDungeonRaid: dungeons,
		}
	})

	writeProto(w, http.StatusOK, reply)
}

func guildSummaryList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ids := r.Form["id"]

	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.GuildSummaryList {
		summaries := make([]*proto.GuildSummary, 0, len(ids))
		for _, idStr := range ids {
			summary := config.GenerateGuildSummary(player)
			summary.GuildId = u64(idStr)
			summaries = append(summaries, summary)
		}
		return &proto.GuildSummaryList{
			Guilds: summaries,
		}
	})
	writeProto(w, http.StatusOK, reply)
}
