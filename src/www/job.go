package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

func jobDeckChange(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	jobID := u32(r.PostForm.Get("job_id"))
	idx := u32(r.PostForm.Get("idx"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		job := player.Job.List[jobID]
		job.JobDeckIdx = idx
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Job: &proto.StoredJob{
					Add: map[uint32]*puser.Job{jobID: job},
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func jobDeckSet(w http.ResponseWriter, r *http.Request) {
	var deck puser.JobDeck
	if !readProto(r, &deck) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		player.JobDeck.List[deck.Id] = &deck
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				JobDeck:    &proto.StoredJobDeck{Add: map[uint64]*puser.JobDeck{deck.Id: &deck}},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}
