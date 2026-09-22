package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/options"
	"example.com/brave-revival/src/proto/pmaster"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type integerValue struct {
	negative  bool
	magnitude uint64
}

func isIntegerKind(kind protoreflect.Kind) bool {
	switch kind {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return true
	default:
		return false
	}
}

func (v integerValue) String() string {
	if v.negative {
		return fmt.Sprintf("-%d", v.magnitude)
	}
	return fmt.Sprint(v.magnitude)
}

func getInteger(message protoreflect.Message, field protoreflect.FieldDescriptor) (integerValue, error) {
	switch field.Kind() {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		value := message.Get(field).Int()
		if value < 0 {
			return integerValue{negative: true, magnitude: uint64(-(value + 1)) + 1}, nil
		}
		return integerValue{magnitude: uint64(value)}, nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return integerValue{magnitude: message.Get(field).Uint()}, nil
	default:
		return integerValue{}, fmt.Errorf("field %s is %s, not an integer", field.FullName(), field.Kind())
	}
}

type foreignKey struct {
	sourceTable protoreflect.FieldDescriptor
	sourceField protoreflect.FieldDescriptor
	target      string
	targetIDs   map[integerValue]struct{}
}

func collectResourceIDs(resources config.Resources) map[integerValue]struct{} {
	ids := make(map[integerValue]struct{})
	for _, catalog := range resources {
		if catalog == nil {
			continue
		}
		for id := range catalog.Resource {
			ids[integerValue{magnitude: uint64(id)}] = struct{}{}
		}
	}
	return ids
}

func collectForeignKeys(master protoreflect.Message, resources config.Resources) ([]foreignKey, error) {
	allFields := master.Descriptor().Fields()
	targets := make(map[string]map[integerValue]struct{})
	resourceIDs := collectResourceIDs(resources)
	var foreignKeys []foreignKey

	for i := range allFields.Len() {
		sourceTable := allFields.Get(i)
		sourceFields := sourceTable.Message().Fields()
		for j := range sourceFields.Len() {
			sourceField := sourceFields.Get(j)
			fk := proto.GetExtension(sourceField.Options(), options.E_Fk).(string)
			if fk == "resources" {
				if !isIntegerKind(sourceField.Kind()) {
					return nil, fmt.Errorf("field %s is %s, not an integer", sourceField.FullName(), sourceField.Kind())
				}
				foreignKeys = append(foreignKeys, foreignKey{
					sourceTable: sourceTable,
					sourceField: sourceField,
					target:      fk,
					targetIDs:   resourceIDs,
				})
				continue
			}

			targetName, ok := strings.CutPrefix(fk, "master/")
			if !ok {
				continue
			}

			targetIDs := targets[targetName]
			if targetIDs == nil {
				targetTable := allFields.ByName(protoreflect.Name(targetName))
				if targetTable == nil {
					return nil, fmt.Errorf("field %s references unknown master table %q", sourceField.FullName(), targetName)
				}
				targetID := targetTable.Message().Fields().ByName("id")
				if targetID == nil {
					return nil, fmt.Errorf("master table %q referenced by %s has no id field", targetName, sourceField.FullName())
				}

				targetIDs = make(map[integerValue]struct{})
				targetRows := master.Get(targetTable).List()
				for k := range targetRows.Len() {
					value, err := getInteger(targetRows.Get(k).Message(), targetID)
					if err != nil {
						return nil, err
					}
					targetIDs[value] = struct{}{}
				}
				targets[targetName] = targetIDs
			}

			if !isIntegerKind(sourceField.Kind()) {
				return nil, fmt.Errorf("field %s is %s, not an integer", sourceField.FullName(), sourceField.Kind())
			}
			foreignKeys = append(foreignKeys, foreignKey{
				sourceTable: sourceTable,
				sourceField: sourceField,
				target:      fk,
				targetIDs:   targetIDs,
			})
		}
	}

	return foreignKeys, nil
}

type missingReference struct {
	foreignKey foreignKey
	value      integerValue
	count      int
	firstRow   int
}

func lintMaster(all *pmaster.All, resources config.Resources) error {
	master := all.ProtoReflect()
	foreignKeys, err := collectForeignKeys(master, resources)
	if err != nil {
		return err
	}

	missingByKey := make(map[string]*missingReference)
	var missing []*missingReference
	for _, fk := range foreignKeys {
		rows := master.Get(fk.sourceTable).List()
		for i := range rows.Len() {
			value, err := getInteger(rows.Get(i).Message(), fk.sourceField)
			if err != nil {
				return err
			}
			if value == (integerValue{}) {
				continue
			}
			if _, ok := fk.targetIDs[value]; ok {
				continue
			}

			key := fmt.Sprintf("%s.%s:%s", fk.sourceTable.Name(), fk.sourceField.Name(), value.String())
			entry := missingByKey[key]
			if entry == nil {
				entry = &missingReference{foreignKey: fk, value: value, firstRow: i + 1}
				missingByKey[key] = entry
				missing = append(missing, entry)
			}
			entry.count++
		}
	}

	if len(missing) == 0 {
		return nil
	}

	lines := make([]string, len(missing))
	for i, entry := range missing {
		lines[i] = fmt.Sprintf(
			"master/%s.%s references missing id %s in %s (%d row(s), first at row %d)",
			entry.foreignKey.sourceTable.Name(), entry.foreignKey.sourceField.Name(), entry.value.String(),
			entry.foreignKey.target, entry.count, entry.firstRow,
		)
	}
	return fmt.Errorf("found %d missing foreign key value(s):\n%s", len(missing), strings.Join(lines, "\n"))
}

func run() error {
	slog.Info("Welcome")

	cfg, err := config.ReadFile(os.Args[1])
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	master, err := config.LoadMaster(cfg.DBDir)
	if err != nil {
		return fmt.Errorf("failed to load master: %w", err)
	}

	resources, err := config.LoadResources(cfg.DBDir)
	if err != nil {
		return fmt.Errorf("failed to load resources: %w", err)
	}

	if err = lintMaster(master, resources); err != nil {
		return err
	}

	slog.Info("Success!")
	return nil
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)

	if len(os.Args) < 2 {
		fmt.Println("Usage: ./dblint <config.json>")
		os.Exit(1)
	}
	if err := run(); err != nil {
		fmt.Println("Error", err)
		os.Exit(1)
	}
}
