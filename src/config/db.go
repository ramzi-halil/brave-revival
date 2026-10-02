package config

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"example.com/brave-revival/src/proto/options"
	"example.com/brave-revival/src/proto/pmaster"
	"golang.org/x/sync/errgroup"
	gproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Resources = map[string]*pmaster.Resources

func LoadResources(dbDir string) (Resources, error) {
	path := filepath.Join(dbDir, "resources.csv")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open resources CSV %q: %w", path, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.FieldsPerRecord = 5

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("cannot read resources CSV: %w", err)
	}

	m := make([]*pmaster.Resources, 2)
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
			for i := range 2 {
				hash := record[i+1]
				size, err := strconv.ParseUint(record[i+3], 10, 32)
				if err != nil {
					return nil, fmt.Errorf("resource CSV contains invalid size: %w", err)
				}
				m[i].Resource[uint32(id)] = &pmaster.ResourceInfo{Hash: hash, Size: uint32(size)}
			}

		case io.EOF:
			result := make(Resources, len(m))
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
	type table struct {
		path          string
		rows          protoreflect.List
		rowDescriptor protoreflect.MessageDescriptor
	}
	tables := make([]table, allFields.Len())

	for i := range allFields.Len() {
		allField := allFields.Get(i)
		tables[i] = table{
			path:          filepath.Join(dbDir, string(allField.Name())+".csv"),
			rows:          allMessage.Mutable(allField).List(),
			rowDescriptor: allField.Message(),
		}
	}

	var group errgroup.Group
	group.SetLimit(runtime.GOMAXPROCS(0))
	for _, table := range tables {
		group.Go(func() error {
			return loadMasterCSV(table.path, table.rows, table.rowDescriptor)
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
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
	reader.Comment = '#'

	type column struct {
		field    protoreflect.FieldDescriptor
		datetime bool
	}
	fields := make([]column, len(header))
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
		fields[i] = column{
			field:    field,
			datetime: field.Kind() == protoreflect.StringKind && gproto.GetExtension(field.Options(), options.E_Datetime).(bool),
		}
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
			value, err := parseMasterValue(raw, fields[i].field, fields[i].datetime)
			if err != nil {
				return fmt.Errorf("master CSV %q record %d column %q: %w", path, recordNumber, fields[i].field.Name(), err)
			}
			row.Set(fields[i].field, value)
		}
		rows.Append(protoreflect.ValueOfMessage(row))
	}
}

func parseMasterValue(raw string, field protoreflect.FieldDescriptor, datetime bool) (protoreflect.Value, error) {
	if datetime && raw != "" {
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
