package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
)

func battleTowerSweep(w http.ResponseWriter, _ *http.Request) {
	clearParam := &proto.BattleClearParam{
		JobId: 1,
		JobParameter: &proto.BaseParameter{},
	}

	writeProto(w, http.StatusOK, &proto.TowerSweepResponse{
		Before: clearParam,
		After:  clearParam,
	})
}
