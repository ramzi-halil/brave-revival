package config

import (
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

const DummyPlayerID = 100

var DummyJob = &puser.Job{
	PlayerId:        DummyPlayerID,
	JobId:           1,
	Level:           999,
	Exp:             999_999_999,
	JobDeckIdx:      1,
	JobEquipmentIdx: 1,
}

var DummyPlayer = &proto.PlayerDetail{
	Player: &puser.Player{
		Id:                    DummyPlayerID,
		WalletId:              "none",
		Status:                "open",
		TutorialStep:          7,
		Nickname:              "Dummy Player",
		Comment:               "Dummy Player's Comment",
		JobId:                 1,
		TitleId:               91706,
		InventorySlots:        999,
		CharacterSlots:        999,
		AccessorySlots:        999,
		Lupi:                  888_888_888_888,
		GuildCoin:             2222,
		Yell:                  333_333,
		AgitoCoin:             444_444,
		InteriorCoin:          555_555,
		BlueSkillPoint:        66_666_666,
		RedSkillPoint:         777_777,
		AchievementRank:       50,
		AchievementPoint:      7_777_777,
		FavoriteEquipmentId_1: 571029947,
		MaxDamage:             999_999_999_999,
		TotalYell:             345_678,
		TotalLogin:            1234,
	},
	CurrentJob: DummyJob,
	Jobs:       []*puser.Job{DummyJob},
	Power:      99_999_999,
}
