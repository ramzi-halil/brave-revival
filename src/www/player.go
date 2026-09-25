package www

import (
	"net/http"
	"os"
	"log/slog"

	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
	"google.golang.org/protobuf/encoding/protojson"
)

func playerList(w http.ResponseWriter, r *http.Request) {
	player := getHandler(r).player
	writeProto(w, http.StatusOK, &proto.PlayerList{
		Players:          []*puser.Player{player.Player},
		HighestJobLv:     player.CurrentJob.Level,
		BackgroundStatus: &proto.BackgroundStatus{},
	})
}

func playerLoad(w http.ResponseWriter, _ *http.Request) {
	data, err := os.ReadFile("./load-sample.json")
	if err != nil {
		slog.Error("failed to read player data", "error", err)
		http.Error(w, "failed to read player data", http.StatusInternalServerError)
		return
	}

	message := &proto.Nocontent{}
	if err := (protojson.UnmarshalOptions{}).Unmarshal(data, message); err != nil {
		slog.Error("failed to parse player data", "error", err)
		http.Error(w, "failed to parse player data", http.StatusInternalServerError)
		return
	}

	writeProto(w, http.StatusOK, message)
}
