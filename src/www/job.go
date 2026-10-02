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
	if deck.Id == 0 {
		deck.Id = uint64(deck.JobId*100 + deck.Idx)
	}

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		player.JobDeck.List[deck.Id] = &deck
		job := player.Job.List[deck.JobId]
		job.JobDeckIdx = deck.Idx
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Job: &proto.StoredJob{
					Add: map[uint32]*puser.Job{deck.JobId: job},
				},
				JobDeck: &proto.StoredJobDeck{Add: map[uint64]*puser.JobDeck{deck.Id: &deck}},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func jobGroupLabelChange(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	jobID := u32(r.PostForm.Get("job_id"))
	idx := u32(r.PostForm.Get("idx"))
	label := r.PostForm.Get("label")

	jobDeckGroupID := uint64(jobID*100 + idx)
	jobDeckGroup := &puser.JobDeckGroup{
		Id:    jobDeckGroupID,
		Idx:   idx,
		JobId: jobID,
		Label: label,
	}

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		jobDeckGroup.PlayerId = player.Player.Id
		player.JobDeckGroup.List[jobDeckGroupID] = jobDeckGroup
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				JobDeckGroup: &proto.StoredJobDeckGroup{
					Add: map[uint64]*puser.JobDeckGroup{jobDeckGroupID: jobDeckGroup},
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func jobDeckLabelChange(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	jobID := u32(r.PostForm.Get("job_id"))
	idx := u32(r.PostForm.Get("idx"))
	label := r.PostForm.Get("label")

	jobDeckID := uint64(jobID*100 + idx)

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		jobDeck := player.JobDeck.List[jobDeckID]
		jobDeck.Label = label
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				JobDeck: &proto.StoredJobDeck{
					Add: map[uint64]*puser.JobDeck{jobDeckID: jobDeck},
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}
