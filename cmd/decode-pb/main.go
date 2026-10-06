package main

import (
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"

	_ "example.com/brave-revival/src/proto/crownotify"
	_ "example.com/brave-revival/src/proto/crowparty"
	_ "example.com/brave-revival/src/proto/prealtime"
	_ "example.com/brave-revival/src/proto/proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func goLiteral(v reflect.Value, depth int) string {
	indent := strings.Repeat("\t", depth)
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return "nil"
		}
		value := goLiteral(v.Elem(), depth)
		if v.Kind() == reflect.Interface {
			return value
		}
		if v.Elem().Kind() == reflect.Struct {
			return "&" + value
		}
		return fmt.Sprintf("func() %s { v := %s(%s); return &v }()", v.Type(), v.Elem().Type(), value)
	case reflect.Struct:
		var fields []string
		for i := range v.NumField() {
			field := v.Type().Field(i)
			if field.IsExported() && !v.Field(i).IsZero() {
				fields = append(fields, indent+"\t"+field.Name+": "+goLiteral(v.Field(i), depth+1)+",\n")
			}
		}
		if len(fields) == 0 {
			return v.Type().String() + "{}"
		}
		return v.Type().String() + "{\n" + strings.Join(fields, "") + indent + "}"
	case reflect.Slice, reflect.Map:
		if v.IsNil() {
			return "nil"
		}
		if v.Len() == 0 {
			return v.Type().String() + "{}"
		}
		var elements []string
		if v.Kind() == reflect.Map {
			iter := v.MapRange()
			for iter.Next() {
				elements = append(elements, indent+"\t"+goLiteral(iter.Key(), depth+1)+": "+goLiteral(iter.Value(), depth+1)+",\n")
			}
		} else {
			for i := range v.Len() {
				elements = append(elements, indent+"\t"+goLiteral(v.Index(i), depth+1)+",\n")
			}
		}
		return v.Type().String() + "{\n" + strings.Join(elements, "") + indent + "}"
	case reflect.Float32, reflect.Float64:
		var special string
		switch {
		case math.IsNaN(v.Float()):
			special = "math.NaN()"
		case math.IsInf(v.Float(), 1):
			special = "math.Inf(1)"
		case math.IsInf(v.Float(), -1):
			special = "math.Inf(-1)"
		}
		if special != "" {
			return v.Type().String() + "(" + special + ")"
		}
	case reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint8, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	}
	return fmt.Sprintf("%#v", v.Interface())
}

func decode(out io.Writer, messageName, input string, format string) error {
	messageType, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(messageName))
	if err != nil {
		return fmt.Errorf("unknown message %q: %w", messageName, err)
	}
	data, err := os.ReadFile(input)
	if err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	message := messageType.New().Interface()
	if err := proto.Unmarshal(data, message); err != nil {
		return fmt.Errorf("decode %s: %w", messageName, err)
	}
	switch format {
	case "go":
		_, err = fmt.Fprintln(out, goLiteral(reflect.ValueOf(message), 0))
	case "json":
		res, err := (protojson.MarshalOptions{Multiline: true}).Marshal(message)
		if err != nil {
			return fmt.Errorf("marshal %s to JSON: %w", messageName, err)
		}
		_, err = out.Write(res)
	default:
		return fmt.Errorf("unknown output format %q", format)
	}
	return err
}

func main() {
	message := flag.String("m", "", "fully qualified Protobuf message name")
	input := flag.String("i", "", "input binary Protobuf file")
	format := flag.String("f", "go", "output format (go/json)")
	flag.Parse()
	if *message == "" || *input == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Usage: decode-pb -m <Message.Type> -i <input.pb> [-f <go|json>]")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if err := decode(os.Stdout, *message, *input, *format); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
