package config

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"example.com/brave-revival/src/proto/options"
	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
	gproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
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

func LoadMaster(dbDir string) (*pmaster.All, error) {
	all := &pmaster.All{}
	allMessage := all.ProtoReflect()
	allFields := allMessage.Descriptor().Fields()

	for i := range allFields.Len() {
		allField := allFields.Get(i)
		path := filepath.Join(dbDir, string(allField.Name())+".csv")
		if err := loadMasterCSV(path, allMessage.Mutable(allField).List(), allField.Message()); err != nil {
			return nil, err
		}
	}

	return all, nil
}

func loadMasterCSV(path string, rows protoreflect.List, rowDescriptor protoreflect.MessageDescriptor) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open master CSV %q: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("cannot read header from master CSV %q: %w", path, err)
	}
	reader.FieldsPerRecord = len(header)
	reader.ReuseRecord = true

	fields := make([]protoreflect.FieldDescriptor, len(header))
	seen := make(map[protoreflect.Name]struct{}, len(header))
	for i, name := range header {
		field := rowDescriptor.Fields().ByName(protoreflect.Name(name))
		if field == nil {
			return fmt.Errorf("master CSV %q has unknown column %q", path, name)
		}
		if _, ok := seen[field.Name()]; ok {
			return fmt.Errorf("master CSV %q has duplicate column %q", path, name)
		}
		seen[field.Name()] = struct{}{}
		fields[i] = field
	}
	if len(seen) != rowDescriptor.Fields().Len() {
		for i := range rowDescriptor.Fields().Len() {
			field := rowDescriptor.Fields().Get(i)
			if _, ok := seen[field.Name()]; !ok {
				return fmt.Errorf("master CSV %q is missing column %q", path, field.Name())
			}
		}
	}

	for recordNumber := 2; ; recordNumber++ {
		record, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("cannot read record %d from master CSV %q: %w", recordNumber, path, err)
		}

		row := rows.NewElement().Message()
		for i, raw := range record {
			value, err := parseMasterValue(raw, fields[i])
			if err != nil {
				return fmt.Errorf("master CSV %q record %d column %q: %w", path, recordNumber, fields[i].Name(), err)
			}
			row.Set(fields[i], value)
		}
		rows.Append(protoreflect.ValueOfMessage(row))
	}
}

func parseMasterValue(raw string, field protoreflect.FieldDescriptor) (protoreflect.Value, error) {
	if field.Kind() == protoreflect.StringKind && raw != "" && gproto.GetExtension(field.Options(), options.E_Datetime).(bool) {
		parsed, err := time.ParseInLocation("2006-01-02T15:04:05", raw, StandardTimeZone)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("invalid datetime %q: %w", raw, err)
		}
		raw = FormatTime(parsed)
	}

	switch field.Kind() {
	case protoreflect.BoolKind:
		value, err := strconv.ParseBool(raw)
		return protoreflect.ValueOfBool(value), err
	case protoreflect.EnumKind:
		value, err := strconv.ParseInt(raw, 10, 32)
		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(value)), err
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		value, err := strconv.ParseInt(raw, 10, 32)
		return protoreflect.ValueOfInt32(int32(value)), err
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		value, err := strconv.ParseInt(raw, 10, 64)
		return protoreflect.ValueOfInt64(value), err
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		value, err := strconv.ParseUint(raw, 10, 32)
		return protoreflect.ValueOfUint32(uint32(value)), err
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		value, err := strconv.ParseUint(raw, 10, 64)
		return protoreflect.ValueOfUint64(value), err
	case protoreflect.FloatKind:
		value, err := strconv.ParseFloat(raw, 32)
		return protoreflect.ValueOfFloat32(float32(value)), err
	case protoreflect.DoubleKind:
		value, err := strconv.ParseFloat(raw, 64)
		return protoreflect.ValueOfFloat64(value), err
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(raw), nil
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte(raw)), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported protobuf kind %s", field.Kind())
	}
}
