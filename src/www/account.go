package www

import (
	"net/http"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/proto"
)

func accountExist(w http.ResponseWriter, r *http.Request) {
	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.PlayerExist {
		return &proto.PlayerExist{
			PlayerSummary:    config.GeneratePlayerSummary(player),
			WorldDescription: "Brave Revival",
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func accountAuthorize(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusOK, &proto.AccountAuthorize{
		Token: "eyJhbGciOiJub25lIn0.eyJleHAiOjk5OTk5OTk5OTk5fQ.",
	})
}

func accountCertificate(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusOK, &proto.AccountCertificate{
		Version: &pmaster.Version{
			Resource: 1,
			Master:   1,
		},
	})
}

func accountInheritPassword(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusOK, &proto.AuthorizeInheritPassword{
		Code: "12345678",
	})
}
