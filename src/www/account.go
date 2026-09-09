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
