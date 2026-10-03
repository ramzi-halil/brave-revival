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
