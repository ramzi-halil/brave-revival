package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/pmisc"
)

func arenaSeason(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusOK, &proto.ArenaSeasonResponse{
		CurrentSeason: &pmisc.ArenaSeason{
			Id: 139,
			OpenDate: "2",
			CloseDate: "ffffffff",
		},
		PreSeason: &pmisc.ArenaSeason{
			Id: 138,
			OpenDate: "1",
			CloseDate: "2",
		},
		PrePreSeason: &pmisc.ArenaSeason{
			Id: 137,
			OpenDate: "0",
			CloseDate: "1",
		},
	})
}

func arenaJobdeckSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	jobDeckId := u64(r.PostForm.Get("job_deck_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		player.Arena.JobDeckId = jobDeckId
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Arena:      player.Arena,
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}
