package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/pmisc"
)

func missionGuildPersonalList(w http.ResponseWriter, r *http.Request) {
	writeProto(w, http.StatusOK, &proto.GuildPersonalMissionResult{
		List: &pmisc.GuildPersonalMissionList{},
	})
}

func missionGuildSharedList(w http.ResponseWriter, r *http.Request) {
	writeProto(w, http.StatusOK, &proto.GuildSharedMissionResult{
		List: &pmisc.GuildSharedMissionList{},
	})
}
