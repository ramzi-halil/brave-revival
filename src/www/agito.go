package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/pmisc"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

func fieldTop(w http.ResponseWriter, r *http.Request) {
	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.FieldTopResponse {
		return &proto.FieldTopResponse{
			StoredData: &proto.StoredData{
				Generation:    player.Generation,
				Player:        player.Player,
				AgitoAp:       &proto.StoredAgitoAp{Add: player.AgitoAp.List},
				AgitoItemArea: &proto.StoredAgitoItemArea{Add: player.AgitoItemArea.List},
			},
			SharedMissionList:       &pmisc.GuildSharedMissionList{},
			PersonalMissionList:     &pmisc.GuildPersonalMissionList{},
			WeeklyMissionReward:     &puser.GuildWeeklyMissionRewardList{},
			FacilityList:            &proto.GuildFacilityList{},
			BackgroundBattleReward:  &proto.BattleBackgroundReward{},
			AgitoVisitorReturn:      &proto.AgitoVisitorReturn{},
			LastAgitoReceivedGoodAt: "0",
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func agitoFurnitureSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	itemIDs := r.PostForm["item_ids"]

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		s := player.AgitoFurnitureSetting.List[1]
		s.WallPaperItemId = u32(itemIDs[0])
		s.FloorBoardItemId = u32(itemIDs[1])
		s.TableSetItemId = u32(itemIDs[2])
		s.SpecialFloorItemId = u32(itemIDs[3])
		s.WallMediumItemId_1 = u32(itemIDs[4])
		s.WallMediumItemId_2 = u32(itemIDs[5])
		s.WallMediumItemId_3 = u32(itemIDs[6])
		s.WallMediumItemId_4 = u32(itemIDs[7])
		s.WallMediumItemId_5 = u32(itemIDs[8])
		s.WallSmallItemId_1 = u32(itemIDs[9])
		s.WallSmallItemId_2 = u32(itemIDs[10])
		s.WallSmallItemId_3 = u32(itemIDs[11])
		s.WallSmallItemId_4 = u32(itemIDs[12])
		s.WallSmallItemId_5 = u32(itemIDs[13])
		s.WallSmallItemId_6 = u32(itemIDs[14])
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				AgitoFurnitureSetting: &proto.StoredAgitoFurnitureSetting{
					Add: map[uint32]*puser.AgitoFurnitureSetting{1: s},
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func agitoItemSet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	areaIDMasked := u32(r.PostForm.Get("agito_item_area_id")) | 0x10000
	itemID := u32(r.PostForm.Get("item_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		area := player.AgitoItemArea.List[areaIDMasked]
		area.ItemId = itemID
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				AgitoItemArea: &proto.StoredAgitoItemArea{
					Add: map[uint32]*puser.AgitoItemArea{areaIDMasked: area},
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}
