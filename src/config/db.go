package config

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"example.com/brave-revival/src/proto/pmaster"
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

type Resources = map[string]*pmaster.Resources

func LoadResources(f io.Reader) (Resources, error) {
	reader := csv.NewReader(f)
	reader.FieldsPerRecord = 3

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("cannot read resources CSV: %w", err)
	}

	m := make([]*pmaster.Resources, len(header)-1)
	for i := range m {
		m[i] = &pmaster.Resources{
			Resource: make(map[uint32]*pmaster.ResourceInfo),
		}
	}

	reader.ReuseRecord = true

	for {
		record, err := reader.Read()
		switch err {
		case nil:
			id, err := strconv.ParseUint(record[0], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("resource CSV contains invalid ID: %w", err)
			}
			for i, hash := range record[1:] {
				m[i].Resource[uint32(id)] = &pmaster.ResourceInfo{Hash: hash}
			}

		case io.EOF:
			result := make(Resources, len(header)-1)
			for i, res := range m {
				result[header[i+1]] = res
			}
			return result, nil

		default:
			return nil, fmt.Errorf("cannot read resources CSV: %w", err)
		}
	}
}
