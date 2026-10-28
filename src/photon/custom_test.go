package photon

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomDataRoundTrip(t *testing.T) {
	hit := HitEffect{TargetId: -2, TargetIndex: 3, Type: OperationTypeDamageHp, HitRating: HitRatingCritical, AttrEffect: -1, Special: -123456789, RelativeValue: 987654321, Interval: 1.5}
	unit := SkillUnit{HitEffects: []HitEffect{hit}}
	character := CharacterResponse{
		Id: -3, Index: 2, Name: "勇者", Type: 1, BossType: 2, DeadCameraType: 3, IsRaid: true,
		DataId: 123, Level: 40, JobId: 5, CurrentHp: 1000, MaxHp: 2000, AwakeningLevel: 6,
		BreakInfo: BreakInfo{CurrentBreakPoint: 100, MaxBreakPoint: 500, BreakRecoverTime: 2.5, CurrentRecoveryTime: 1.25},
		Form:      -1, NormalActiveSkillId: 456,
		ActiveSkills: []ActiveSkill{{Id: 7, Level: 8, Number: 9, CoolTime: 3.5, RemainCoolTime: 1.5}},
		Avatar:       Avatar{EquipmentIds: []uint32{10, 11}, StylishFlag: 1, FrontEnhancement: 12},
	}
	deck := DeckResponse{
		ActorNr: -5, Id: -4, ActiveIndex: 1, Characters: []CharacterResponse{character},
		Potions: []Potion{{Id: 123, Num: -2}}, HpPotionSetting: PotionSetting{Auto: true, Rank: -1}, JobId: 2, MaxLevel: 100,
	}
	team := GvgTeam{TeamNr: 2, IsFortressDestroyed: true, WithinVisibilityBuilding: true, ShrineOccupancy: 3, GuildId: math.MaxUint64, GuildLevel: 4, GuildFacilities: []GuildFacility{{Id: 5, Level: 6}}}
	player := GvgPlayer{PlayerId: math.MaxUint64, HpRates: []float32{0.5, 1}, MpRates: []float32{0.25}}
	ranking := RankingRow{Name: "player", Rank: -1, Score: math.MaxUint64, IsSelf: true}
	values := []CustomData{
		&SkillInputResponse{Index: 2, SkillId: -123, Result: SkillInputResultMpShortage},
		&SkillActivateResponse{Id: -1, Index: 2, TargetId: -3, SkillId: 123, CastTime: 1.5},
		&SkillInvokeResponse{Id: -1, Index: 2, SkillId: 123, SkillUnits: []SkillUnit{unit}},
		&SkillCoolResponse{Id: -1, Index: 2, SkillId: 123, CoolTime: 1.5, InitialCoolTime: 2.5},
		&SkillSealResponse{Id: -1, Index: 2, SkillNumbers: []uint32{123, math.MaxUint32}, AbortSkillId: -3, CoolTime: 1.5},
		&SkillCancelResponse{Id: -1, Index: 2, SkillId: 123},
		&PotionInvokeResponse{Id: -1, ItemId: 123, CoolTime: 1.5, HitEffects: []HitEffect{hit}},
		&PotionSettingInput{Type: PotionSettingTypeMpRank, Value: 3},
		&unit,
		&hit,
		&deck,
		&character,
		&BattleEnterResponse{
			StageId: 123, CloseTs: -456, StartWave: 2, ContinueCount: 3, Seed: -789,
			Waves: []WaveResponse{{WaveId: 1, SubId: 2}}, Teams: [][]DeckResponse{{deck}, {}, {deck, deck}},
			Resume: BattleResumeResponse{
				TargetResume:         TargetResumeResponse{AllyTargetId: -1, EnemyTargetId: -2},
				PotionCommandResumes: []PotionCommandResumeResponse{{Type: 3, RemainCoolTime: 1.5}},
				SwitchCommandResume:  SwitchCommandResumeResponse{IsReserve: true, DeckIndex: 4},
				SkillCommandResume:   SkillCommandResumeResponse{SealNumbers: []uint32{5, 6}},
			},
			RaidId: 7, PartyId: "party", GvgBattleFieldId: 8, GvgBuildingCategory: 9, Risk: 10,
		},
		&WaveResponse{WaveId: -1, SubId: 2},
		&NextWaveResponse{Wave: 2, Enemies: []DeckResponse{deck}},
		&MidwayReadyResponse{Success: true, State: 2, ElapsedTime: 1.5, NextWave: NextWaveResponse{Wave: 3, Enemies: []DeckResponse{deck}}, CharacterIds: []int16{-1, 2}},
		&DropRewardResponse{CharacterId: -1, BoxColors: []uint8{2, 3}},
		&BattleExitResponse{RoomId: "room", ResultType: BattleResultTypeWinBossHpRate},
		&TreasureBattleStatus{BossLupi: -1, EnemyLupi: 2, CriticalLupi: 3},
		&PossessionReleaseResponse{Id: -1, Index: 2, SkillUnit: unit, Possessions: []PossessionType{PossessionTypeA, PossessionTypeC}},
		&SwitchResponse{Id: -1, Index: 2, Ailments: []int32{-3, 4}},
		&PassiveInvokeResponse{Id: -1, Index: 2, PassiveSkillId: -3, EquipmentId: 4},
		&Profile{Id: -1, JobId: 2, JobLevel: 3, MaxLevel: 4, Power: 5, Lines: []ProfileLine{{DeckIndex: 6, Equipments: []ProfileEquipment{{Category: 7, EquipmentId: 8, Rarity: 9, Level: 10, Enhancement: 11, AwakeningLevel: 12}}}}},
		&BreakRecoveryTimeResponse{BreakRecoveryTimeInfos: []BreakRecoverTimeInfo{{Id: -1, Index: 2, CurrentRecoveryTime: 1.5}}},
		&GvgEntryRequest{StageId: 1, GvgBattleFieldId: 2, GvgBuildingCategory: 3, GvgBattleType: GvgBattleTypePvMDefense, GvgWaveId: 4, RaidId: 5, FirstAttackPlayerId: math.MaxUint64, RankingEntryName: "entry", Teams: []GvgTeam{team}, Players: []GvgPlayer{player}, StartBattleTime: "start", MatchTime: 6},
		&GvgIntrudeRequest{Team: team, Player: player},
		&team,
		&player,
		&GuildFacility{Id: math.MaxUint32, Level: 2},
		&RaidStatsResponse{Exists: true, BossHp: math.MaxUint64 - 1, BossMaxHp: math.MaxUint64, Rankings: []RankingRow{ranking}},
		&ranking,
	}
	for _, value := range values {
		t.Run(value.CustomCode().String(), func(t *testing.T) {
			encoded, err := marshalCustomData(value)
			require.NoError(t, err)
			decoded, err := unmarshalCustomData(value.CustomCode(), encoded)
			require.NoError(t, err)
			require.Equal(t, value, decoded)
			for n := 0; n < len(encoded); n++ {
				_, err := unmarshalCustomData(value.CustomCode(), encoded[:n])
				require.ErrorIsf(t, err, ErrMalformedMessage, "truncated at %d", n)
			}
			_, err = unmarshalCustomData(value.CustomCode(), append(encoded, 0))
			require.ErrorIs(t, err, ErrMalformedMessage)
		})
	}
}

func TestCustomDataEncoding(t *testing.T) {
	for _, tt := range []struct {
		value CustomData
		bytes []byte
	}{
		{&SkillInputResponse{Index: 0x12, SkillId: -2, Result: SkillInputResultFailed}, []byte{0x12, 0xfe, 0xff, 0xff, 0xff, 3}},
		{&SkillActivateResponse{Id: 0x1234, Index: 0x56, TargetId: -2, SkillId: 0x12345678, CastTime: 1.5}, []byte{0x34, 0x12, 0x56, 0xfe, 0xff, 0x78, 0x56, 0x34, 0x12, 0, 0, 0xc0, 0x3f}},
		{&RankingRow{Name: "勇", Rank: -2, Score: 0x0123456789abcdef, IsSelf: true}, []byte{3, 0xe5, 0x8b, 0x87, 0xfe, 0xff, 0xff, 0xff, 0xef, 0xcd, 0xab, 0x89, 0x67, 0x45, 0x23, 1, 1}},
		{&DropRewardResponse{CharacterId: -2, BoxColors: []byte{3, 4}}, []byte{0xfe, 0xff, 2, 0, 3, 4}},
		{&PossessionReleaseResponse{Id: -2, Index: 3, SkillUnit: SkillUnit{HitEffects: []HitEffect{}}, Possessions: []PossessionType{PossessionTypeA, PossessionTypeC}}, []byte{0xfe, 0xff, 3, 0, 0, 2, 0, 1, 3}},
	} {
		t.Run(tt.value.CustomCode().String(), func(t *testing.T) {
			encoded, err := marshalCustomData(tt.value)
			require.NoError(t, err)
			require.Equal(t, tt.bytes, encoded)
			decoded, err := unmarshalCustomData(tt.value.CustomCode(), tt.bytes)
			require.NoError(t, err)
			require.Equal(t, tt.value, decoded)
		})
	}
}

func TestCustomStringLength(t *testing.T) {
	for _, tt := range []struct {
		length int
		prefix []byte
	}{
		{0, []byte{0}},
		{127, []byte{0x7f}},
		{128, []byte{0x80, 1}},
		{16383, []byte{0xff, 0x7f}},
		{16384, []byte{0x80, 0x80, 1}},
	} {
		value := &BattleExitResponse{RoomId: strings.Repeat("a", tt.length), ResultType: BattleResultTypeWin}
		encoded, err := marshalCustomData(value)
		require.NoError(t, err)
		require.Equal(t, append(append(tt.prefix, []byte(value.RoomId)...), byte(value.ResultType)), encoded)
		decoded, err := unmarshalCustomData(value.CustomCode(), encoded)
		require.NoError(t, err)
		require.Equal(t, value, decoded)
	}
	for _, data := range [][]byte{{0x80}, bytes.Repeat([]byte{0xff}, 10), {0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 1}, {2, 'a'}} {
		_, err := unmarshalCustomData(CustomCodeBattleExitResponse, data)
		require.ErrorIs(t, err, ErrMalformedMessage)
	}
}

func TestCustomArrayLength(t *testing.T) {
	for _, length := range []int{0, 256, math.MaxUint16} {
		value := &DropRewardResponse{BoxColors: make([]byte, length)}
		encoded, err := marshalCustomData(value)
		require.NoError(t, err)
		require.Equal(t, []byte{byte(length), byte(length >> 8)}, encoded[2:4])
		decoded, err := unmarshalCustomData(value.CustomCode(), encoded)
		require.NoError(t, err)
		require.Equal(t, value, decoded)
	}
	encoded, err := marshalCustomData(&DropRewardResponse{BoxColors: make([]byte, math.MaxUint16+1)})
	require.ErrorIs(t, err, ErrMessageTooLarge)
	require.Nil(t, encoded)
	encoded, err = marshalCustomData(&DropRewardResponse{})
	require.NoError(t, err)
	require.Equal(t, []byte{0, 0, 0, 0}, encoded)
}

func TestCustomDataInvalid(t *testing.T) {
	for _, value := range []CustomData{nil, (*SkillInputResponse)(nil)} {
		_, err := marshalCustomData(value)
		require.ErrorIs(t, err, ErrUnsupportedObject)
	}
	_, err := unmarshalCustomData(CustomCode(255), nil)
	require.ErrorIs(t, err, ErrUnsupportedObject)
}
