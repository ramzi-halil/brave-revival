package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/pmisc"
)

func fieldTop(w http.ResponseWriter, r *http.Request) {
	writeProto(w, http.StatusOK, &proto.FieldTopResponse{
		SharedMissionList: &pmisc.GuildSharedMissionList{},
		PersonalMissionList: &pmisc.GuildPersonalMissionList{},
		WeeklyMissionReward: &pmisc.GuildWeeklyMissionRewardList{},
		FacilityList: &proto.GuildFacilityList{},
		BackgroundBattleReward: &proto.BattleBackgroundReward{},
		AgitoVisitorReturn: &proto.AgitoVisitorReturn{},
	})
}
