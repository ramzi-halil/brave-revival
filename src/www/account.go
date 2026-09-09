package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
)

func accountExist(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusOK, &proto.PlayerExist{
		PlayerSummary: &proto.PlayerSummary{
			PlayerId: 100,
			Nickname: "Dummy Player",
			JobId:    1,
			JobLevel: 999,
			Power:    99_999_999,
		},
		WorldDescription: "Dummy World",
	})
}
