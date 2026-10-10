package www

import (
	"maps"
	"net/http"
	"slices"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

func playerList(w http.ResponseWriter, r *http.Request) {
	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.PlayerList {
		return &proto.PlayerList{
			Players:          []*puser.Player{player.Player},
			HighestJobLv:     780,
			BackgroundStatus: &proto.BackgroundStatus{},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func playerLoad(w http.ResponseWriter, r *http.Request) {
	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.Nocontent {
		return &proto.Nocontent{StoredData: player}
	})
	writeProto(w, http.StatusOK, reply)
}

func playerDetail(w http.ResponseWriter, r *http.Request) {
	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.PlayerDetail {
		job := player.Job.List[player.Player.JobId]
		var currentDeck *puser.JobDeck
		for _, deck := range player.JobDeck.List {
			if deck.JobId == job.JobId && deck.Idx == job.JobDeckIdx {
				currentDeck = deck
				break
			}
		}

		return &proto.PlayerDetail{
			Player:         player.Player,
			CurrentJob:     job,
			Jobs:           slices.Collect(maps.Values(player.Job.List)),
			CurrentJobDeck: currentDeck,
			Equipments:     slices.Collect(maps.Values(player.Equipment.List)),
			BaseParameter:  &proto.BaseParameter{},
			GuildSummary:   config.GenerateGuildSummary(player),
			GuildMember:    &proto.GuildMember{},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func playerChangeFavorite(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	query := r.PostForm

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		player.Player.FavoriteEquipmentId_1 = u64(query.Get("equipment_id_1"))
		player.Player.FavoriteEquipmentId_1Costume = u64(query.Get("equipment_id_1_costume"))
		player.Player.FavoriteEquipmentId_1StylishHead = u64(query.Get("equipment_id_1_stylish_head"))
		player.Player.FavoriteEquipmentId_1StylishBody = u64(query.Get("equipment_id_1_stylish_body"))
		player.Player.FavoriteEquipmentId_2 = u64(query.Get("equipment_id_2"))
		player.Player.FavoriteEquipmentId_2Costume = u64(query.Get("equipment_id_2_costume"))
		player.Player.FavoriteEquipmentId_2StylishHead = u64(query.Get("equipment_id_2_stylish_head"))
		player.Player.FavoriteEquipmentId_2StylishBody = u64(query.Get("equipment_id_2_stylish_body"))
		player.Player.FavoriteEquipmentId_3 = u64(query.Get("equipment_id_3"))
		player.Player.FavoriteEquipmentId_3Costume = u64(query.Get("equipment_id_3_costume"))
		player.Player.FavoriteEquipmentId_3StylishHead = u64(query.Get("equipment_id_3_stylish_head"))
		player.Player.FavoriteEquipmentId_3StylishBody = u64(query.Get("equipment_id_3_stylish_body"))
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func playerChangeNickname(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	nickname := r.PostForm.Get("nickname")

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		player.Player.Nickname = nickname
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func playerChangeComment(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	comment := r.PostForm.Get("comment")

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		player.Player.Comment = comment
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func playerChangeJob(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	jobId := u32(r.PostForm.Get("job_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		player.Player.JobId = jobId
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func titleSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	titleId := u32(r.PostForm.Get("title_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		player.Player.TitleId = titleId
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func playerSummaryList(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	playerIds := r.Form["player_id"]

	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.PlayerSummaryList {
		players := make([]*proto.PlayerSummary, 0, len(playerIds))
		for _, playerId := range playerIds {
			p := config.GeneratePlayerSummary(player)
			p.PlayerId = u64(playerId)
			players = append(players, p)
		}
		return &proto.PlayerSummaryList{
			Players: players,
		}
	})
	writeProto(w, http.StatusOK, reply)
}
