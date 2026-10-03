package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
)

func gvgHistoryList(w http.ResponseWriter, r *http.Request) {
	battleFieldID := u32(r.URL.Query().Get("battle_field_id"))

	writeProto(w, http.StatusOK, &proto.GvgHistoryList{
		BattleFieldId: battleFieldID,
	})
}

func gvgBidTopFieldList(w http.ResponseWriter, r *http.Request) {
	gvgID := u32(r.URL.Query().Get("gvg_id"))

	endTimeMap := make(map[uint32]string)
	handler := getHandler(r)
	handler.masterLock.Lock()
	for _, field := range handler.master.GvgBattleField {
		if field.GvgId == gvgID {
			endTimeMap[field.Id] = "0"
		}
	}
	handler.masterLock.Unlock()

	writeProto(w, http.StatusOK, &proto.GvgBidTopFieldList{
		EndTimeMap: endTimeMap,
	})
}

// TODO: does not work yet!
/*
func gvgPracticeTop(w http.ResponseWriter, r *http.Request) {
	writeProto(w, http.StatusOK, &proto.GvgPracticeInfo{
		GvgPracticeMatching: &pmisc.GvgPracticeMatching{
			EntryAt: "0",
			MatchingAt: "0",
		},
		GvgPracticeBattle: &pmisc.GvgPracticeBattle{
			StartBattleDate: "0",
		},
		StopSchedule: []*pmisc.GvgPracticeStopSchedule{{
			FromDate: "0",
			ToDate:   "0",
		}},
	})
}
*/
