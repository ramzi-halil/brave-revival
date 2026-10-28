package photon

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
)

type CustomData interface {
	CustomCode() CustomCode
}

var customDataTypes = map[CustomCode]reflect.Type{
	CustomCodeSkillInputResponse:        reflect.TypeFor[SkillInputResponse](),
	CustomCodeSkillActivateResponse:     reflect.TypeFor[SkillActivateResponse](),
	CustomCodeSkillInvokeResponse:       reflect.TypeFor[SkillInvokeResponse](),
	CustomCodeSkillCoolResponse:         reflect.TypeFor[SkillCoolResponse](),
	CustomCodeSkillSealResponse:         reflect.TypeFor[SkillSealResponse](),
	CustomCodeSkillCancelResponse:       reflect.TypeFor[SkillCancelResponse](),
	CustomCodePotionInvokeResponse:      reflect.TypeFor[PotionInvokeResponse](),
	CustomCodePotionSettingInput:        reflect.TypeFor[PotionSettingInput](),
	CustomCodeSkillUnit:                 reflect.TypeFor[SkillUnit](),
	CustomCodeHitEffect:                 reflect.TypeFor[HitEffect](),
	CustomCodeDeckResponse:              reflect.TypeFor[DeckResponse](),
	CustomCodeCharacterResponse:         reflect.TypeFor[CharacterResponse](),
	CustomCodeBattleEnterResponse:       reflect.TypeFor[BattleEnterResponse](),
	CustomCodeWaveResponse:              reflect.TypeFor[WaveResponse](),
	CustomCodeNextWaveResponse:          reflect.TypeFor[NextWaveResponse](),
	CustomCodeMidwayReadyResponse:       reflect.TypeFor[MidwayReadyResponse](),
	CustomCodeDropRewardResponse:        reflect.TypeFor[DropRewardResponse](),
	CustomCodeBattleExitResponse:        reflect.TypeFor[BattleExitResponse](),
	CustomCodeTreasureBattleStatus:      reflect.TypeFor[TreasureBattleStatus](),
	CustomCodePossessionReleaseResponse: reflect.TypeFor[PossessionReleaseResponse](),
	CustomCodeSwitchResponse:            reflect.TypeFor[SwitchResponse](),
	CustomCodePassiveInvokeResponse:     reflect.TypeFor[PassiveInvokeResponse](),
	CustomCodeProfile:                   reflect.TypeFor[Profile](),
	CustomCodeBreakRecoveryTimeResponse: reflect.TypeFor[BreakRecoveryTimeResponse](),
	CustomCodeGvgEntryRequest:           reflect.TypeFor[GvgEntryRequest](),
	CustomCodeGvgIntrudeRequest:         reflect.TypeFor[GvgIntrudeRequest](),
	CustomCodeGvgTeam:                   reflect.TypeFor[GvgTeam](),
	CustomCodeGvgPlayer:                 reflect.TypeFor[GvgPlayer](),
	CustomCodeGuildFacility:             reflect.TypeFor[GuildFacility](),
	CustomCodeRaidStatsResponse:         reflect.TypeFor[RaidStatsResponse](),
	CustomCodeRankingRow:                reflect.TypeFor[RankingRow](),
}

func marshalCustomData(data CustomData) ([]byte, error) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return nil, protocolError(ErrUnsupportedObject, "custom data must be a non-nil struct")
	}
	var buf bytes.Buffer
	if err := writeCustomValue(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func unmarshalCustomData(code CustomCode, data []byte) (CustomData, error) {
	t, ok := customDataTypes[code]
	if !ok {
		return nil, protocolError(ErrUnsupportedObject, fmt.Sprintf("unknown custom code %d", code))
	}
	v := reflect.New(t)
	r := bytes.NewReader(data)
	if err := readCustomValue(r, v.Elem()); err != nil {
		return nil, err
	}
	if r.Len() != 0 {
		return nil, protocolError(ErrMalformedMessage, "trailing custom data")
	}
	return v.Interface().(CustomData), nil
}

func writeCustomValue(w *bytes.Buffer, v reflect.Value) error {
	if !v.CanInterface() {
		return protocolError(ErrUnsupportedObject, "unexported custom data field")
	}
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := writeCustomValue(w, v.Field(i)); err != nil {
				return fmt.Errorf("%s.%s: %w", v.Type(), v.Type().Field(i).Name, err)
			}
		}
	case reflect.Slice, reflect.Array:
		if v.Len() > math.MaxUint16 {
			return ErrMessageTooLarge
		}
		_ = binary.Write(w, binary.LittleEndian, uint16(v.Len()))
		for i := 0; i < v.Len(); i++ {
			if err := writeCustomValue(w, v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.String:
		var length [binary.MaxVarintLen64]byte
		n := binary.PutUvarint(length[:], uint64(v.Len()))
		w.Write(length[:n])
		w.WriteString(v.String())
	case reflect.Bool, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return binary.Write(w, binary.LittleEndian, v.Interface())
	default:
		return protocolError(ErrUnsupportedObject, fmt.Sprintf("unsupported custom data type %s", v.Type()))
	}
	return nil
}

func readCustomValue(r *bytes.Reader, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if err := readCustomValue(r, v.Field(i)); err != nil {
				return fmt.Errorf("%s.%s: %w", v.Type(), v.Type().Field(i).Name, err)
			}
		}
	case reflect.Slice, reflect.Array:
		var length uint16
		if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
			return malformedRead(err)
		}
		if v.Kind() == reflect.Slice {
			v.Set(reflect.MakeSlice(v.Type(), int(length), int(length)))
		} else if v.Len() != int(length) {
			return protocolError(ErrMalformedMessage, "custom array length mismatch")
		}
		for i := 0; i < v.Len(); i++ {
			if err := readCustomValue(r, v.Index(i)); err != nil {
				return err
			}
		}
	case reflect.String:
		length, err := binary.ReadUvarint(r)
		if err != nil {
			return malformedRead(err)
		}
		data, err := readBytes(r, length)
		if err != nil {
			return err
		}
		v.SetString(string(data))
	case reflect.Bool, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return malformedRead(binary.Read(r, binary.LittleEndian, v.Addr().Interface()))
	default:
		return protocolError(ErrUnsupportedObject, fmt.Sprintf("unsupported custom data type %s", v.Type()))
	}
	return nil
}

type SkillInputResult uint8

const (
	SkillInputResultSucceeded SkillInputResult = iota
	SkillInputResultReserved
	SkillInputResultCanceled
	SkillInputResultFailed
	SkillInputResultMpShortage
	SkillInputResultNotExistTarget
)

type SkillInputResponse struct {
	Index   uint8
	SkillId int32
	Result  SkillInputResult
}

func (*SkillInputResponse) CustomCode() CustomCode {
	return CustomCodeSkillInputResponse
}

type SkillActivateResponse struct {
	Id       int16
	Index    uint8
	TargetId int16
	SkillId  int32
	CastTime float32
}

func (*SkillActivateResponse) CustomCode() CustomCode {
	return CustomCodeSkillActivateResponse
}

type OperationType uint8

const (
	OperationTypeSkillNone OperationType = iota
	OperationTypeDamageHp
	OperationTypeDamageMp
	OperationTypeHealHp
	OperationTypeHealMp
	OperationTypeAbsorptionHp
	OperationTypeAbsorptionMp
	OperationTypeSelfHarm
	OperationTypeBuff
	OperationTypeDebuff
	OperationTypeBuffDebuffCure
	OperationTypeAilment
	OperationTypeCure
	OperationTypeGuard
	OperationTypeFate
	OperationTypePoison
	OperationTypeProstration
	OperationTypeConfusion
	OperationTypeParalysis
	OperationTypeFrozen
	OperationTypeDotParalysis
	OperationTypeHpRegeneration
	OperationTypeMpRegeneration
	OperationTypeRevive
	OperationTypeCounter
	OperationTypeCounterattack
	OperationTypeBuffRelease
	OperationTypeFormChange
	OperationTypeDisappear
	OperationTypePossession
	OperationTypeBossLupi
	OperationTypeEnemyLupi
	OperationTypeCriticalLupi
	OperationTypeDamageBreakGauge
	OperationTypeHealBreakGauge
	OperationTypeSyncRecoveryTime
)

type HitRating uint8

const (
	HitRatingMiss HitRating = iota
	HitRatingResistance
	HitRatingNormal
	HitRatingCritical
)

type HitEffect struct {
	TargetId      int16
	TargetIndex   uint8
	Type          OperationType
	HitRating     HitRating
	AttrEffect    int8
	Special       int64
	RelativeValue int64
	Interval      float32
}

func (*HitEffect) CustomCode() CustomCode {
	return CustomCodeHitEffect
}

type SkillUnit struct {
	HitEffects []HitEffect
}

func (*SkillUnit) CustomCode() CustomCode {
	return CustomCodeSkillUnit
}

type SkillInvokeResponse struct {
	Id         int16
	Index      uint8
	SkillId    int32
	SkillUnits []SkillUnit
}

func (*SkillInvokeResponse) CustomCode() CustomCode {
	return CustomCodeSkillInvokeResponse
}

type SkillCoolResponse struct {
	Id              int16
	Index           uint8
	SkillId         int32
	CoolTime        float32
	InitialCoolTime float32
}

func (*SkillCoolResponse) CustomCode() CustomCode {
	return CustomCodeSkillCoolResponse
}

type SkillSealResponse struct {
	Id           int16
	Index        uint8
	SkillNumbers []uint32
	AbortSkillId int32
	CoolTime     float32
}

func (*SkillSealResponse) CustomCode() CustomCode {
	return CustomCodeSkillSealResponse
}

type SkillCancelResponse struct {
	Id      int16
	Index   uint8
	SkillId int32
}

func (*SkillCancelResponse) CustomCode() CustomCode {
	return CustomCodeSkillCancelResponse
}

type PotionInvokeResponse struct {
	Id         int16
	ItemId     uint32
	CoolTime   float32
	HitEffects []HitEffect
}

func (*PotionInvokeResponse) CustomCode() CustomCode {
	return CustomCodePotionInvokeResponse
}

type PotionSettingType uint8

const (
	PotionSettingTypeHpAuto PotionSettingType = iota
	PotionSettingTypeHpRank
	PotionSettingTypeMpAuto
	PotionSettingTypeMpRank
)

type PotionSettingInput struct {
	Type  PotionSettingType
	Value uint8
}

func (*PotionSettingInput) CustomCode() CustomCode {
	return CustomCodePotionSettingInput
}

type BreakInfo struct {
	CurrentBreakPoint   int64
	MaxBreakPoint       int64
	BreakRecoverTime    float32
	CurrentRecoveryTime float32
}

type ActiveSkill struct {
	Id             uint32
	Level          uint32
	Number         uint32
	CoolTime       float32
	RemainCoolTime float32
}

type Avatar struct {
	EquipmentIds     []uint32
	StylishFlag      uint32
	FrontEnhancement uint32
}

type CharacterResponse struct {
	Id                  int16
	Index               uint8
	Name                string
	Type                uint8
	BossType            uint8
	DeadCameraType      uint8
	IsRaid              bool
	DataId              uint32
	Level               int16
	JobId               uint8
	CurrentHp           int64
	MaxHp               int64
	AwakeningLevel      uint32
	BreakInfo           BreakInfo
	Form                int32
	NormalActiveSkillId uint32
	ActiveSkills        []ActiveSkill
	Avatar              Avatar
}

func (*CharacterResponse) CustomCode() CustomCode {
	return CustomCodeCharacterResponse
}

type Potion struct {
	Id  uint32
	Num int32
}

type PotionSetting struct {
	Auto bool
	Rank int32
}

type DeckResponse struct {
	ActorNr         int32
	Id              int16
	ActiveIndex     uint8
	Characters      []CharacterResponse
	Potions         []Potion
	HpPotionSetting PotionSetting
	JobId           uint8
	MaxLevel        uint16
}

func (*DeckResponse) CustomCode() CustomCode {
	return CustomCodeDeckResponse
}

type WaveResponse struct {
	WaveId int32
	SubId  int32
}

func (*WaveResponse) CustomCode() CustomCode {
	return CustomCodeWaveResponse
}

type TargetResumeResponse struct {
	AllyTargetId  int16
	EnemyTargetId int16
}

type PotionCommandResumeResponse struct {
	Type           uint8
	RemainCoolTime float32
}

type SwitchCommandResumeResponse struct {
	IsReserve bool
	DeckIndex uint8
}

type SkillCommandResumeResponse struct {
	SealNumbers []uint32
}

type BattleResumeResponse struct {
	TargetResume         TargetResumeResponse
	PotionCommandResumes []PotionCommandResumeResponse
	SwitchCommandResume  SwitchCommandResumeResponse
	SkillCommandResume   SkillCommandResumeResponse
}

type BattleEnterResponse struct {
	StageId             uint32
	CloseTs             int64
	StartWave           uint8
	ContinueCount       uint8
	Seed                int32
	Waves               []WaveResponse
	Teams               [][]DeckResponse
	Resume              BattleResumeResponse
	RaidId              uint32
	PartyId             string
	GvgBattleFieldId    uint32
	GvgBuildingCategory uint8
	Risk                uint32
}

func (*BattleEnterResponse) CustomCode() CustomCode {
	return CustomCodeBattleEnterResponse
}

type NextWaveResponse struct {
	Wave    uint8
	Enemies []DeckResponse
}

func (*NextWaveResponse) CustomCode() CustomCode {
	return CustomCodeNextWaveResponse
}

type MidwayReadyResponse struct {
	Success      bool
	State        uint8
	ElapsedTime  float32
	NextWave     NextWaveResponse
	CharacterIds []int16
}

func (*MidwayReadyResponse) CustomCode() CustomCode {
	return CustomCodeMidwayReadyResponse
}

type DropRewardResponse struct {
	CharacterId int16
	BoxColors   []uint8
}

func (*DropRewardResponse) CustomCode() CustomCode {
	return CustomCodeDropRewardResponse
}

type BattleResultType uint8

const (
	BattleResultTypeLose BattleResultType = iota
	BattleResultTypeWin
	BattleResultTypeWinBossHpRate
)

type BattleExitResponse struct {
	RoomId     string
	ResultType BattleResultType
}

func (*BattleExitResponse) CustomCode() CustomCode {
	return CustomCodeBattleExitResponse
}

type TreasureBattleStatus struct {
	BossLupi     int32
	EnemyLupi    int32
	CriticalLupi int32
}

func (*TreasureBattleStatus) CustomCode() CustomCode {
	return CustomCodeTreasureBattleStatus
}

type PossessionType uint8

const (
	PossessionTypeNone PossessionType = iota
	PossessionTypeA
	PossessionTypeB
	PossessionTypeC
)

type PossessionReleaseResponse struct {
	Id          int16
	Index       uint8
	SkillUnit   SkillUnit
	Possessions []PossessionType
}

func (*PossessionReleaseResponse) CustomCode() CustomCode {
	return CustomCodePossessionReleaseResponse
}

type SwitchResponse struct {
	Id       int16
	Index    uint8
	Ailments []int32
}

func (*SwitchResponse) CustomCode() CustomCode {
	return CustomCodeSwitchResponse
}

type PassiveInvokeResponse struct {
	Id             int16
	Index          uint8
	PassiveSkillId int32
	EquipmentId    uint32
}

func (*PassiveInvokeResponse) CustomCode() CustomCode {
	return CustomCodePassiveInvokeResponse
}

type ProfileEquipment struct {
	Category       uint8
	EquipmentId    uint32
	Rarity         uint8
	Level          uint16
	Enhancement    uint8
	AwakeningLevel uint32
}

type ProfileLine struct {
	DeckIndex  uint8
	Equipments []ProfileEquipment
}

type Profile struct {
	Id       int16
	JobId    uint8
	JobLevel uint16
	MaxLevel uint16
	Power    uint32
	Lines    []ProfileLine
}

func (*Profile) CustomCode() CustomCode {
	return CustomCodeProfile
}

type BreakRecoverTimeInfo struct {
	Id                  int16
	Index               uint8
	CurrentRecoveryTime float32
}

type BreakRecoveryTimeResponse struct {
	BreakRecoveryTimeInfos []BreakRecoverTimeInfo
}

func (*BreakRecoveryTimeResponse) CustomCode() CustomCode {
	return CustomCodeBreakRecoveryTimeResponse
}

type GvgBattleType uint8

const (
	GvgBattleTypePvP GvgBattleType = iota + 1
	GvgBattleTypePvE
	GvgBattleTypePvMNeutral
	GvgBattleTypePvMDefense
)

type GuildFacility struct {
	Id    uint32
	Level uint32
}

func (*GuildFacility) CustomCode() CustomCode {
	return CustomCodeGuildFacility
}

type GvgTeam struct {
	TeamNr                   uint8
	IsFortressDestroyed      bool
	WithinVisibilityBuilding bool
	ShrineOccupancy          uint8
	GuildId                  uint64
	GuildLevel               uint32
	GuildFacilities          []GuildFacility
}

func (*GvgTeam) CustomCode() CustomCode {
	return CustomCodeGvgTeam
}

type GvgPlayer struct {
	PlayerId uint64
	HpRates  []float32
	MpRates  []float32
}

func (*GvgPlayer) CustomCode() CustomCode {
	return CustomCodeGvgPlayer
}

type GvgEntryRequest struct {
	StageId             uint32
	GvgBattleFieldId    uint32
	GvgBuildingCategory uint8
	GvgBattleType       GvgBattleType
	GvgWaveId           uint32
	RaidId              uint32
	FirstAttackPlayerId uint64
	RankingEntryName    string
	Teams               []GvgTeam
	Players             []GvgPlayer
	StartBattleTime     string
	MatchTime           uint32
}

func (*GvgEntryRequest) CustomCode() CustomCode {
	return CustomCodeGvgEntryRequest
}

type GvgIntrudeRequest struct {
	Team   GvgTeam
	Player GvgPlayer
}

func (*GvgIntrudeRequest) CustomCode() CustomCode {
	return CustomCodeGvgIntrudeRequest
}

type RankingRow struct {
	Name   string
	Rank   int32
	Score  uint64
	IsSelf bool
}

func (*RankingRow) CustomCode() CustomCode {
	return CustomCodeRankingRow
}

type RaidStatsResponse struct {
	Exists    bool
	BossHp    uint64
	BossMaxHp uint64
	Rankings  []RankingRow
}

func (*RaidStatsResponse) CustomCode() CustomCode {
	return CustomCodeRaidStatsResponse
}
