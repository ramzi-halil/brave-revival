package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
)

func battleTowerSweep(w http.ResponseWriter, _ *http.Request) {
	clearParam := &proto.BattleClearParam{
		JobId:        1,
		JobParameter: &proto.BaseParameter{},
	}

	writeProto(w, http.StatusOK, &proto.TowerSweepResponse{
		Before: clearParam,
		After:  clearParam,
	})
}

func riskDungeonGetRisk(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	playerIds := r.PostForm["player_ids"]

	maps := make(map[uint64]uint32, len(playerIds))
	for _, playerIdStr := range playerIds {
		maps[u64(playerIdStr)] = 1
	}

	writeProto(w, http.StatusOK, &proto.RiskDungeonGetRiskResponse{
		RiskMap:          maps,
		ChallengeUnitMap: maps,
	})
}
