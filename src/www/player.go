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
