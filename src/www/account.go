package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
)

func accountExist(w http.ResponseWriter, r *http.Request) {
	player := getHandler(r).player
	writeProto(w, http.StatusOK, &proto.PlayerExist{
		PlayerSummary: &proto.PlayerSummary{
			PlayerId: player.Player.Id,
			Nickname: player.Player.Nickname,
			JobId:    player.Player.JobId,
			JobLevel: 999,
			Power:    999,
		},
		WorldDescription: "Brave Revival",
	})
}

func accountAuthorize(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusOK, &proto.AccountAuthorize{
		Token: "eyJhbGciOiJub25lIn0.eyJleHAiOjk5OTk5OTk5OTk5fQ.",
	})
}

func accountCertificate(w http.ResponseWriter, r *http.Request) {
	writeProto(w, http.StatusOK, &proto.AccountCertificate{
		Version: getHandler(r).master.Version[0],
	})
}
