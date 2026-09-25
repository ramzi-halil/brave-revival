package config

import (
	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/pmisc"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

const defaultPlayerID = 100

func GenerateDefaultPlayer(master *pmaster.All) *proto.StoredData {
	storedEquipments := make(map[uint64]*puser.Equipment, len(master.Equipment))
	for _, equipment := range master.Equipment {
		storedEquipments[uint64(equipment.Id)] = &puser.Equipment{
			Id:                uint64(equipment.Id),
			PlayerId:          defaultPlayerID,
			EquipmentId:       equipment.Id,
			Rarity:            1,
			Level:             1,
			WeaponSkillLevel1: 1,
			WeaponSkillLevel2: 1,
			WeaponSkillLevel3: 1,
			AcquiredAt:        "0",
		}
	}

	jobDecks := make(map[uint64]*puser.JobDeck, 5)
	for i := 1; i <= 5; i++ {
		jobDecks[uint64(i)] = &puser.JobDeck{
			Id:             uint64(i),
			Idx:            1,
			PlayerId:       defaultPlayerID,
			JobId:          uint32(i),
			Line1MainFront: 1007140,
			HpUseAt:        40,
			HpUseOrder:     1,
		}
	}

	jobs := make(map[uint32]*puser.Job, 5)
	for i := uint32(1); i <= 5; i++ {
		jobs[i] = &puser.Job{
			PlayerId:        defaultPlayerID,
			JobId:           i,
			Level:           999,
			JobDeckIdx:      1,
			JobEquipmentIdx: 1,
		}
	}

	agitoItemArea := make(map[uint32]*puser.AgitoItemArea, len(master.AgitoItemArea))
	for _, area := range master.AgitoItemArea {
		agitoItemArea[area.Id] = &puser.AgitoItemArea{
			PlayerId:        defaultPlayerID,
			RoomNumber:      1,
			AgitoItemAreaId: area.Id,
			LineupReturnAt1: "0",
			LineupReturnAt2: "0",
			LineupReturnAt3: "0",
		}
	}

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
			AchievementRank:       50,
			FavoriteEquipmentId_1: 1007140,
			FavoriteEquipmentId_2: 1044030,
			LastLoginAt:           "0",
			NameChangedAt:         "0",
			NewbieShopOpenedAt:    "0",
			ComeBackExpiredAt:     "ffffffff",
			OpenedAt:              "0",
		},
		Currency: &proto.Currency{
			RedOrb:      1200,
			FreeBlueOrb: 1200,
			TotalOrb:    2400,
		},
		Equipment: &proto.StoredEquipment{List: storedEquipments},
		Arena: &puser.Arena{
			PlayerId:              defaultPlayerID,
			JobDeckId:             1,
			DailyAcquiredAt:       "0",
			WeeklyAcquiredAt:      "0",
			DailyRewardReceivedAt: "0",
		},
		JobDeck:       &proto.StoredJobDeck{List: jobDecks},
		Job:           &proto.StoredJob{List: jobs},
		AgitoItemArea: &proto.StoredAgitoItemArea{List: agitoItemArea},
		AgitoFurnitureSetting: &proto.StoredAgitoFurnitureSetting{
			List: map[uint32]*puser.AgitoFurnitureSetting{
				1: &puser.AgitoFurnitureSetting{
					PlayerId:         defaultPlayerID,
					RoomNumber:       1,
					WallPaperItemId:  35476,
					FloorBoardItemId: 35477,
					TableSetItemId:   35684,
				},
			},
		},
		AgitoAp: &proto.StoredAgitoAp{
			List: map[uint32]*puser.AgitoAp{
				1: &puser.AgitoAp{
					PlayerId:      defaultPlayerID,
					RoomNumber:    1,
					ItemId:        34005,
					Ap:            9999,
					NextLotteryAt: "ffffffff",
				},
			},
		},
		Sample:                      &proto.StoredSample{},
		Setting:                     &puser.Setting{},
		Item:                        &proto.StoredItem{},
		Anima:                       &proto.StoredAnima{},
		AnimaArea:                   &proto.StoredAnimaArea{},
		Rune:                        &proto.StoredRune{},
		Elixir:                      &proto.StoredElixir{},
		Contents:                    &proto.StoredContents{},
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
		FunctionalTutorial:          &proto.StoredFunctionalTutorial{},
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
		Vip:                         &puser.Vip{},
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
		JobDeckGroup:                &proto.StoredJobDeckGroup{},
		GvgPracticeReward:           &proto.StoredGvgPracticeReward{},
		ShopSpecialSale:             &proto.StoredShopSpecialSale{},
	}
}
