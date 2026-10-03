package www

import (
	"net/http"
	"strings"

	"example.com/brave-revival/src/proto/pcommon"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func calculateStylishFlag(costume, stylishBody, stylishWeapon, stylishDevice uint64) uint32 {
	var flag uint32
	if costume != 0 {
		flag |= 2
	}
	if stylishBody != 0 {
		flag |= 4
	}
	if stylishWeapon != 0 {
		flag |= 8
	}
	if stylishDevice != 0 {
		flag |= 16
	}
	return flag
}

func jobDeckEquipmentRemoveAll(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentID := u64(r.PostForm.Get("equipment_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		for _, jobDeck := range player.JobDeck.List {
			if (jobDeck.Line1MainFront == 0 || jobDeck.Line1MainFront == equipmentID) &&
				(jobDeck.Line2MainFront == 0 || jobDeck.Line2MainFront == equipmentID) &&
				(jobDeck.Line3MainFront == 0 || jobDeck.Line3MainFront == equipmentID) {
				return &proto.Nocontent{Error: &pcommon.Error{Code: 16002, Level: 3}}
			}
		}

		changedDecks := make(map[uint64]*puser.JobDeck)
		for id, jobDeck := range player.JobDeck.List {
			deck := jobDeck.ProtoReflect()
			deck.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
				if field.Kind() == protoreflect.Uint64Kind && strings.HasPrefix(string(field.Name()), "line") && value.Uint() == equipmentID {
					deck.Clear(field)
					changedDecks[id] = jobDeck
				}
				return true
			})
		}
		for _, jobDeck := range changedDecks {
			jobDeck.Line1StylishFlag = calculateStylishFlag(jobDeck.Line1MainCostume, jobDeck.Line1MainStylishBody, jobDeck.Line1MainStylishWeapon, jobDeck.Line1MainStylishDevice)
			jobDeck.Line2StylishFlag = calculateStylishFlag(jobDeck.Line2MainCostume, jobDeck.Line2MainStylishBody, jobDeck.Line2MainStylishWeapon, jobDeck.Line2MainStylishDevice)
			jobDeck.Line3StylishFlag = calculateStylishFlag(jobDeck.Line3MainCostume, jobDeck.Line3MainStylishBody, jobDeck.Line3MainStylishWeapon, jobDeck.Line3MainStylishDevice)
		}

		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				JobDeck:    &proto.StoredJobDeck{Add: player.JobDeck.List},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}
