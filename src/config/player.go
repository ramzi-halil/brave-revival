package config

import (
	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/pmisc"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

const defaultPlayerID = 100

// 2026-09-09 21:30 JST
const defaultTimestamp = "6aa15148"

// generateStoredEquipment generates the list of stored equipments for the player,
// such that the player has 1 copy of each equipment.
// The 2nd field is an arbitrary vanguard ID for use in the default deck.
func generateStoredEquipment(master *pmaster.All) (*proto.StoredEquipment, uint64) {
	defaultTargetParamOption := map[uint32]uint32{0: 0}
	for _, targetParamOption := range master.TargetParamOption {
		defaultTargetParamOption[targetParamOption.GroupId] = targetParamOption.Id
	}

	storedEquipments := make(map[uint64]*puser.Equipment, len(master.Equipment))
	var anyEquipmentID uint64
	for _, equipment := range master.Equipment {
		rarity := uint32(1)
		if equipment.EquipmentCategory <= 7 {
			rarity = 6
		}
		paramOption := defaultTargetParamOption[equipment.TargetParamOptionGroupId]

		storedEquipments[uint64(equipment.Id)] = &puser.Equipment{
			Id:                uint64(equipment.Id),
			PlayerId:          defaultPlayerID,
			EquipmentId:       equipment.Id,
			Rarity:            rarity,
			Level:             1,
			WeaponSkillLevel1: 1,
			WeaponSkillLevel2: 1,
			WeaponSkillLevel3: 1,
			ParamOption1:      paramOption,
			ParamOption2:      paramOption,
			ParamOption3:      paramOption,
			AcquiredAt:        defaultTimestamp,
		}
		if anyEquipmentID < 10000 && equipment.EquipmentCategory == 1 {
			anyEquipmentID = uint64(equipment.Id)
		}
	}
	return &proto.StoredEquipment{List: storedEquipments}, anyEquipmentID
}

func generateStoredJobDeck(master *pmaster.All, anyEquipmentID uint64) *proto.StoredJobDeck {
	jobDecks := make(map[uint64]*puser.JobDeck, 5)
	for i := uint32(1); i <= 5; i++ {
		jobDeckID := uint64(i*100 + 1)
		jobDecks[jobDeckID] = &puser.JobDeck{
			Id:             jobDeckID,
			Idx:            1,
			PlayerId:       defaultPlayerID,
			JobId:          i,
			Line1MainFront: anyEquipmentID,
			HpUseAt:        40,
			HpUseOrder:     1,
		}
	}
	return &proto.StoredJobDeck{List: jobDecks}
}

func generateStoredJob(master *pmaster.All) *proto.StoredJob {
	jobs := make(map[uint32]*puser.Job, 5)
	for i := uint32(1); i <= 5; i++ {
		jobs[i] = &puser.Job{
			PlayerId:        defaultPlayerID,
			JobId:           i,
			Level:           780,
			JobDeckIdx:      1,
			JobEquipmentIdx: 1,
		}
	}
	return &proto.StoredJob{List: jobs}
}

func generateStoredAgitoItemArea(master *pmaster.All) *proto.StoredAgitoItemArea {
	agitoItemArea := make(map[uint32]*puser.AgitoItemArea, len(master.AgitoItemArea))
	for _, area := range master.AgitoItemArea {
		agitoItemArea[area.Id|0x10000] = &puser.AgitoItemArea{
			PlayerId:        defaultPlayerID,
			RoomNumber:      1,
			AgitoItemAreaId: area.Id,
			LineupReturnAt1: "0",
			LineupReturnAt2: "0",
			LineupReturnAt3: "0",
		}
	}
	return &proto.StoredAgitoItemArea{List: agitoItemArea}
}

func generateStoredFunctionTutorial(master *pmaster.All) *proto.StoredFunctionalTutorial {
	tutorials := make(map[uint32]*puser.FunctionalTutorial, len(master.FunctionalTutorial))
	for _, tutorial := range master.FunctionalTutorial {
		step := uint32(2)
		if tutorial.Id == 11 {
			step = 7
		}
		tutorials[tutorial.Id] = &puser.FunctionalTutorial{
			PlayerId:   defaultPlayerID,
			FunctionId: tutorial.Id,
			Step:       step,
			Adid:       "054f522be8b4cc4a7063159715950950",
		}
	}
	return &proto.StoredFunctionalTutorial{List: tutorials}
}

func generateStoredContents(master *pmaster.All) *proto.StoredContents {
	areaContents := make(map[uint32]uint32, len(master.Area))
	for _, area := range master.Area {
		areaContents[area.Id] = area.ContentsId
	}

	maxStageByContents := make(map[uint32]uint32, len(master.Contents))
	for _, stage := range master.Stage {
		contentsID, ok := areaContents[stage.AreaId]
		if ok && stage.Id > maxStageByContents[contentsID] {
			maxStageByContents[contentsID] = stage.Id
		}
	}

	contents := make(map[uint32]*puser.Contents, len(master.Contents))
	for _, content := range master.Contents {
		contents[content.Id] = &puser.Contents{
			PlayerId:    defaultPlayerID,
			ContentsId:  content.Id,
			StageId:     maxStageByContents[content.Id],
			LastSweptAt: defaultTimestamp,
		}
	}
	return &proto.StoredContents{List: contents}
}

func generateStoredItem(master *pmaster.All) *proto.StoredItem {
	items := make(map[uint32]*puser.Item)
	for _, item := range master.Item {
		quantity := uint32(9999)
		if item.Type == 36 {
			quantity = 1
		}
		items[item.Id] = &puser.Item{
			PlayerId:   defaultPlayerID,
			ItemId:     item.Id,
			Quantity:   quantity,
			AcquiredAt: defaultTimestamp,
		}
	}
	return &proto.StoredItem{List: items}
}

func GenerateDefaultPlayer(master *pmaster.All) *proto.StoredData {
	equipments, anyEquipmentID := generateStoredEquipment(master)

	return &proto.StoredData{
		Generation: 1,
		Player: &puser.Player{
			Id:                    defaultPlayerID,
			Status:                "open",
			TutorialStep:          7,
			Nickname:              "sample",
			JobId:                 1,
			InventorySlots:        9999,
			CharacterSlots:        9999,
			AccessorySlots:        9999,
			Lupi:                  999_999_999,
			Topaz:                 9999,
			Mileage:               9999,
			AgitoCoin:             9999,
			InteriorCoin:          9999,
			BlueSkillPoint:        999_999_999,
			RedSkillPoint:         999_999_999,
			AchievementRank:       50,
			FavoriteEquipmentId_1: anyEquipmentID,
			LastLoginAt:           defaultTimestamp,
			NameChangedAt:         defaultTimestamp,
			NewbieShopOpenedAt:    defaultTimestamp,
			ComeBackExpiredAt:     "ffffffff",
			OpenedAt:              defaultTimestamp,
		},
		Currency: &proto.Currency{
			RedOrb:      9999,
			FreeBlueOrb: 9999,
			TotalOrb:    19998,
		},
		Arena: &puser.Arena{
			PlayerId:              defaultPlayerID,
			JobDeckId:             101,
			DailyAcquiredAt:       defaultTimestamp,
			WeeklyAcquiredAt:      defaultTimestamp,
			DailyRewardReceivedAt: defaultTimestamp,
		},
		AgitoFurnitureSetting: &proto.StoredAgitoFurnitureSetting{
			List: map[uint32]*puser.AgitoFurnitureSetting{
				1: &puser.AgitoFurnitureSetting{
					PlayerId:         defaultPlayerID,
					RoomNumber:       1,
					WallPaperItemId:  35001,
					FloorBoardItemId: 35002,
					TableSetItemId:   35003,
				},
			},
		},
		AgitoAp: &proto.StoredAgitoAp{
			List: map[uint32]*puser.AgitoAp{
				1: &puser.AgitoAp{
					PlayerId:      defaultPlayerID,
					RoomNumber:    1,
					ItemId:        34001,
					Ap:            9999,
					NextLotteryAt: "ffffffff",
				},
			},
		},
		Vip: &puser.Vip{
			PlayerId: defaultPlayerID,
			Rank:     12,
		},
		JobDeckGroup: &proto.StoredJobDeckGroup{
			List: map[uint64]*puser.JobDeckGroup{},
		},
		Equipment:                   equipments,
		JobDeck:                     generateStoredJobDeck(master, anyEquipmentID),
		Job:                         generateStoredJob(master),
		AgitoItemArea:               generateStoredAgitoItemArea(master),
		FunctionalTutorial:          generateStoredFunctionTutorial(master),
		Contents:                    generateStoredContents(master),
		Item:                        generateStoredItem(master),
		Sample:                      &proto.StoredSample{},
		Setting:                     &puser.Setting{},
		Anima:                       &proto.StoredAnima{},
		AnimaArea:                   &proto.StoredAnimaArea{},
		Rune:                        &proto.StoredRune{},
		Elixir:                      &proto.StoredElixir{},
		JobSkill:                    &proto.StoredJobSkill{},
		ConditionProgress:           &proto.StoredConditionProgress{},
		Achievement:                 &proto.StoredAchievement{},
		DailyMission:                &proto.StoredDailyMission{},
		DailyMissionReward:          &proto.StoredDailyMissionReward{},
		OrderMission:                &proto.StoredOrderMission{},
		OrderMissionReward:          &proto.StoredOrderMissionReward{},
		OrderMissionReroll:          &proto.StoredOrderMissionReroll{},
		BattleMember:                &pmisc.BattleMember{},
		GuildInfo:                   &proto.GuildInfo{},
		MercenaryHire:               &proto.StoredMercenaryHire{},
		MercenaryReward:             &puser.MercenaryReward{},
		AchievementReward:           &puser.AchievementReward{},
		EventMission:                &proto.StoredEventMission{},
		EventMissionReward:          &proto.StoredEventMissionReward{},
		ContentsCondition:           &proto.StoredContentsCondition{},
		AbyssFever:                  &puser.AbyssFever{},
		Mercenary:                   &proto.StoredMercenary{},
		Title:                       &proto.StoredTitle{},
		ShopItem:                    &proto.StoredShopItem{},
		LoginBonus:                  &proto.StoredLoginBonus{},
		EventSugoroku:               &proto.StoredEventSugoroku{},
		Boost:                       &proto.StoredBoost{},
		Block:                       &proto.StoredBlock{},
		Exchange:                    &proto.StoredExchange{},
		ContentsHero:                &proto.StoredContentsHero{},
		ContentsTreasure:            &proto.StoredContentsTreasure{},
		ContentsWeekMonster:         &proto.StoredContentsWeekMonster{},
		BackgroundBattle:            &puser.BackgroundBattle{},
		EventRoulette:               &proto.StoredEventRoulette{},
		AngelBattleWeeklyReward:     &proto.StoredAngelBattleWeeklyReward{},
		AchievementEquipment:        &proto.StoredAchievementEquipment{},
		AchievementEquipmentReceive: &proto.StoredAchievementEquipmentReceive{},
		AchievementEquipmentStamp:   &proto.StoredAchievementEquipmentStamp{},
		EquipmentLiberation:         &proto.StoredEquipmentLiberation{},
		ChatUnreadCategories:        &proto.ChatUnreadCategory{},
		AgitoRelotteryInterval:      &proto.StoredAgitoRelotteryInterval{},
		AgitoVisitor:                &proto.StoredAgitoVisitor{},
		Agito:                       &puser.Agito{},
		AgitoGoodHistory:            &proto.StoredAgitoGoodHistory{},
		GachaHistory:                &proto.StoredGachaHistory{},
		AgitoCountReward:            &proto.StoredAgitoCountReward{},
		SeasonPass:                  &proto.StoredSeasonPass{},
		SeasonPassDailyMission:      &proto.StoredSeasonPassDailyMission{},
		SeasonPassWeeklyMission:     &proto.StoredSeasonPassWeeklyMission{},
		ImportantMissionGroup:       &proto.StoredImportantMissionGroup{},
		ImportantMission:            &proto.StoredImportantMission{},
		SpecialItemProgress:         &proto.StoredSpecialItemProgress{},
		Advertising:                 &proto.StoredAdvertising{},
		PresentBoxInfo:              &proto.StoredPresentBoxInfo{},
		ReliefPoint:                 &puser.ReliefPoint{},
		ReliefPointSending:          &proto.StoredReliefPointSending{},
		ReliefPointReward:           &proto.StoredReliefPointReward{},
		ContentsRiskDungeon:         &proto.StoredContentsRiskDungeon{},
		ContentsClearAncientTowerEx: &proto.StoredContentsClearAncientTowerEx{},
		GvgPracticeReward:           &proto.StoredGvgPracticeReward{},
		ShopSpecialSale:             &proto.StoredShopSpecialSale{},
	}
}
