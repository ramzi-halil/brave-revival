package www

import (
	"maps"
	"net/http"
	"slices"

	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

func playerList(w http.ResponseWriter, r *http.Request) {
	player := getHandler(r).player
	writeProto(w, http.StatusOK, &proto.PlayerList{
		Players:          []*puser.Player{player.Player},
		HighestJobLv:     780,
		BackgroundStatus: &proto.BackgroundStatus{},
	})
}

func playerLoad(w http.ResponseWriter, r *http.Request) {
	player := getHandler(r).player
	writeProto(w, http.StatusOK, &proto.Nocontent{StoredData: player})
}

func playerDetail(w http.ResponseWriter, r *http.Request) {
	player := getHandler(r).player
	writeProto(w, http.StatusOK, &proto.PlayerDetail{
		Player:         player.Player,
		CurrentJob:     player.Job.List[1],
		Jobs:           slices.Collect(maps.Values(player.Job.List)),
		CurrentJobDeck: player.JobDeck.List[1],
		Equipments:     slices.Collect(maps.Values(player.Equipment.List)),
		BaseParameter:  &proto.BaseParameter{},
		GuildSummary:   &proto.GuildSummary{},
		GuildMember:    &proto.GuildMember{},
	})
}

func playerChangeFavorite(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	query := r.PostForm
	handler := getHandler(r)
	player := handler.player
	player.Generation++
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
	handler.savePlayer()
	writeProto(w, http.StatusOK, &proto.Nocontent{
		StoredData: &proto.StoredData{
			Generation: player.Generation,
			Player:     player.Player,
		},
	})
}
