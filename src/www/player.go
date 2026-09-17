package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

func playerList(w http.ResponseWriter, r *http.Request) {
	player := getHandler(r).player
	writeProto(w, http.StatusOK, &proto.PlayerList{
		Players:          []*puser.Player{player.Player},
		HighestJobLv:     player.CurrentJob.Level,
		BackgroundStatus: &proto.BackgroundStatus{},
	})
}

func playerLoad(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-protobuf")
	w.Header().Set("proto-type", "Proto.Nocontent")
	http.ServeFile(w, r, "./load-sample.pb")
}
