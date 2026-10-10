package www

import (
	"math/rand/v2"
	"net/http"

	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/pmisc"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

func fieldTop(w http.ResponseWriter, r *http.Request) {
	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.FieldTopResponse {
		var lineupIDs []uint32
		for _, area := range player.AgitoItemArea.List {
			if area.AgitoVisitorLineupId_1 != 0 {
				lineupIDs = append(lineupIDs, area.AgitoVisitorLineupId_1)
			}
			if area.AgitoVisitorLineupId_2 != 0 {
				lineupIDs = append(lineupIDs, area.AgitoVisitorLineupId_2)
			}
			if area.AgitoVisitorLineupId_3 != 0 {
				lineupIDs = append(lineupIDs, area.AgitoVisitorLineupId_3)
			}
		}

		return &proto.FieldTopResponse{
			StoredData: &proto.StoredData{
				Generation:    player.Generation,
				Player:        player.Player,
				AgitoAp:       &proto.StoredAgitoAp{Add: player.AgitoAp.List},
				AgitoItemArea: &proto.StoredAgitoItemArea{Add: player.AgitoItemArea.List},
			},
			GuildId:                 player.GuildInfo.GuildId,
			GuildName:               player.GuildInfo.GuildName,
			GuildSymbol:             player.GuildInfo.GuildSymbol,
			GuildSymbolFrame:        player.GuildInfo.GuildSymbolFrame,
			GuildSymbolFrameColor:   player.GuildInfo.GuildSymbolFrameColor,
			GuildMemberRole:         player.GuildInfo.GuildMemberRole,
			SharedMissionList:       &pmisc.GuildSharedMissionList{},
			PersonalMissionList:     &pmisc.GuildPersonalMissionList{},
			WeeklyMissionReward:     &puser.GuildWeeklyMissionRewardList{},
			FacilityList:            &pmisc.GuildFacilityList{},
			BackgroundBattleReward:  &proto.BattleBackgroundReward{},
			AgitoVisitorReturn:      &proto.AgitoVisitorReturn{},
			LastAgitoReceivedGoodAt: "0",
			AgitoNewLineupIds:       lineupIDs,
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

func agitoPlayerRecommend(w http.ResponseWriter, r *http.Request) {
	handler := getHandler(r)
	handler.masterLock.Lock()
	var equipments []*pmaster.Equipment
	for _, equipment := range handler.master.Equipment {
		if equipment.EquipmentCategory == 1 || equipment.EquipmentCategory == 2 {
			equipments = append(equipments, equipment)
		}
	}
	var equipmentIDs [3]uint32
	memberIDs := make(map[uint32]bool)
	for len(equipments) > 0 && len(memberIDs) < len(equipmentIDs) {
		i := rand.IntN(len(equipments))
		equipment := equipments[i]
		equipments[i] = equipments[len(equipments)-1]
		equipments = equipments[:len(equipments)-1]
		if memberIDs[equipment.MemberId] {
			continue
		}
		equipmentIDs[len(memberIDs)] = equipment.Id
		memberIDs[equipment.MemberId] = true
	}

	furnitureIDs := make(map[uint32][]uint32)
	for _, furniture := range handler.master.AgitoFurniture {
		furnitureIDs[furniture.Type] = append(furnitureIDs[furniture.Type], furniture.ItemId)
	}
	pickFurniture := func(itemType uint32) uint32 {
		ids := furnitureIDs[itemType]
		if len(ids) == 0 {
			return 0
		}
		return ids[rand.IntN(len(ids))]
	}

	var areas, optionalAreas []*pmaster.AgitoItemArea
	selected := make(map[uint32]bool)
	excluded := make(map[uint32]bool)
	selectArea := func(area *pmaster.AgitoItemArea) {
		areas = append(areas, area)
		selected[area.Id] = true
		excluded[area.ExceptAgitoItemAreaId_1] = true
		excluded[area.ExceptAgitoItemAreaId_2] = true
		excluded[area.ExceptAgitoItemAreaId_3] = true
	}
	for _, area := range handler.master.AgitoItemArea {
		switch area.AgitoItemType {
		case 1, 3:
			selectArea(area)
		case 2, 4:
			optionalAreas = append(optionalAreas, area)
		}
	}
	remaining := map[uint32]int{2: 4, 4: 1}
	for _, i := range rand.Perm(len(optionalAreas)) {
		area := optionalAreas[i]
		if remaining[area.AgitoItemType] == 0 || excluded[area.Id] || selected[area.ExceptAgitoItemAreaId_1] || selected[area.ExceptAgitoItemAreaId_2] || selected[area.ExceptAgitoItemAreaId_3] {
			continue
		}
		selectArea(area)
		remaining[area.AgitoItemType]--
	}

	itemsByType := make(map[uint32][]*pmaster.AgitoItem)
	seenItemIDs := make(map[uint32]bool)
	for _, item := range handler.master.AgitoItem {
		if !seenItemIDs[item.ItemId] {
			itemsByType[item.Type] = append(itemsByType[item.Type], item)
			seenItemIDs[item.ItemId] = true
		}
	}
	lineupIDs := make(map[uint32][]uint32)
	for _, lineup := range handler.master.AgitoVisitorLineup {
		if lineup.EquipmentId != 0 {
			lineupIDs[lineup.LineupId] = append(lineupIDs[lineup.LineupId], lineup.Id)
		}
	}
	pickLineup := func(lineupID uint32) uint32 {
		ids := lineupIDs[lineupID]
		if len(ids) == 0 {
			return 0
		}
		return ids[rand.IntN(len(ids))]
	}
	var itemAreas []*puser.AgitoItemArea
	for _, area := range areas {
		items := itemsByType[area.AgitoItemType]
		if len(items) == 0 {
			continue
		}
		i := rand.IntN(len(items))
		item := items[i]
		itemArea := &puser.AgitoItemArea{
			PlayerId:               200,
			RoomNumber:             1,
			AgitoItemAreaId:        area.Id,
			ItemId:                 item.ItemId,
			AgitoVisitorLineupId_1: pickLineup(item.AgitoVisitorLineupId_1),
			AgitoVisitorLineupId_2: pickLineup(item.AgitoVisitorLineupId_2),
			AgitoVisitorLineupId_3: pickLineup(item.AgitoVisitorLineupId_3),
			LineupReturnAt1:        "0",
			LineupReturnAt2:        "0",
			LineupReturnAt3:        "0",
		}
		var lineups []*uint32
		for _, id := range []*uint32{&itemArea.AgitoVisitorLineupId_1, &itemArea.AgitoVisitorLineupId_2, &itemArea.AgitoVisitorLineupId_3} {
			if *id != 0 {
				lineups = append(lineups, id)
			}
		}
		if len(lineups) > 1 {
			choice := rand.IntN(len(lineups) + 1)
			for i, id := range lineups {
				if choice < len(lineups) && i != choice {
					*id = 0
				}
			}
		}
		itemAreas = append(itemAreas, itemArea)
		items[i] = items[len(items)-1]
		itemsByType[area.AgitoItemType] = items[:len(items)-1]
	}

	reply := &proto.AgitoInfo{
		PlayerId:                    200,
		Nickname:                    "sample #2",
		JobId:                       1,
		JobLevel:                    780,
		PlayerTitleId:               handler.master.Title[rand.IntN(len(handler.master.Title))].Id,
		FavoriteEquipmentId_1:       equipmentIDs[0],
		FavoriteEquipmentId_2:       equipmentIDs[1],
		FavoriteEquipmentId_3:       equipmentIDs[2],
		SelfLastAgitoReceivedGoodAt: "0",
		AgitoRoom: []*proto.AgitoRoom{
			{
				RoomNumber: 1,
				AgitoFurnitureSetting: &puser.AgitoFurnitureSetting{
					PlayerId:           200,
					RoomNumber:         1,
					WallPaperItemId:    pickFurniture(1),
					FloorBoardItemId:   pickFurniture(2),
					TableSetItemId:     pickFurniture(3),
					SpecialFloorItemId: pickFurniture(6),
					WallMediumItemId_1: pickFurniture(7),
					WallMediumItemId_2: pickFurniture(7),
					WallMediumItemId_3: pickFurniture(7),
					WallMediumItemId_4: pickFurniture(7),
					WallMediumItemId_5: pickFurniture(7),
					WallSmallItemId_1:  pickFurniture(8),
					WallSmallItemId_2:  pickFurniture(8),
					WallSmallItemId_3:  pickFurniture(8),
					WallSmallItemId_4:  pickFurniture(8),
					WallSmallItemId_5:  pickFurniture(8),
					WallSmallItemId_6:  pickFurniture(8),
				},
				AgitoItemArea: itemAreas,
			},
		},
	}
	handler.masterLock.Unlock()

	writeProto(w, http.StatusOK, reply)
}

func findEquipmentAndItemByLineupID(master *pmaster.All, lineupID uint32) (equipmentID, itemID uint32) {
	for _, lineup := range master.AgitoVisitorLineup {
		if lineup.Id == lineupID {
			equipmentID = lineup.EquipmentId
			for _, item := range master.AgitoItem {
				switch lineup.LineupId {
				case item.AgitoVisitorLineupId_1, item.AgitoVisitorLineupId_2, item.AgitoVisitorLineupId_3:
					itemID = item.ItemId
					return
				}
			}
		}
	}
	return
}

func agitoItemReceive(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	lineupID := u32(r.PostForm.Get("lineup_id"))

	handler := getHandler(r)
	handler.masterLock.Lock()
	equipmentID, itemID := findEquipmentAndItemByLineupID(handler.master, lineupID)
	handler.masterLock.Unlock()

	writeProto(w, http.StatusOK, &proto.AgitoVisitorRewardReceive{
		StoredData: &proto.StoredData{
			AgitoVisitor: &proto.StoredAgitoVisitor{
				Add: map[uint32]*puser.AgitoVisitor{
					equipmentID: {
						EquipmentId:    equipmentID,
						FirstVisitedAt: "0",
					},
				},
			},
		},
		ReceiveReward: &proto.AgitoVisitorReturnReward{
			ItemId:      itemID,
			EquipmentId: equipmentID,
			ReceiveRewardList: []*proto.RewardInfo{
				{
					TargetType: 22,
					Quantity:   1,
				},
				{
					TargetType: 25,
					Quantity:   1,
				},
			},
		},
	})
}
