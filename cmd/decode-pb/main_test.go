package main

import (
	"bytes"
	"go/parser"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"example.com/brave-revival/src/proto/crowparty"
	"example.com/brave-revival/src/proto/prealtime"
	gameproto "example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name      string
		message   proto.Message
		want      string
		unordered bool
	}{
		{
			name: "nested messages",
			message: &gameproto.FieldTopResponse{
				StoredData: &gameproto.StoredData{
					Generation: 2112495379672953621,
					Player:     &puser.Player{Id: 642214},
				},
			},
			want: `&proto.FieldTopResponse{
	StoredData: &proto.StoredData{
		Generation: 2112495379672953621,
		Player: &puser.Player{
			Id: 642214,
		},
	},
}
`,
		},
		{
			name:      "numeric map",
			unordered: true,
			message: &gameproto.StoredItem{List: map[uint32]*puser.Item{
				10: {ItemId: 10},
				2:  {ItemId: 2},
			}},
			want: `&proto.StoredItem{
	List: map[uint32]*puser.Item{
		2: &puser.Item{
			ItemId: 2,
		},
		10: &puser.Item{
			ItemId: 10,
		},
	},
}
`,
		},
		{
			name:      "string map",
			unordered: true,
			message:   &prealtime.StatNumSub{Nums: map[string]int32{"b": 0, "a": -3}},
			want: `&prealtime.StatNumSub{
	Nums: map[string]int32{
		"a": -3,
		"b": 0,
	},
}
`,
		},
		{
			name:    "repeated strings",
			message: &prealtime.Channel{Channels: []string{"a\n\"b", ""}},
			want: `&prealtime.Channel{
	Channels: []string{
		"a\n\"b",
		"",
	},
}
`,
		},
		{
			name:    "oneof with zero value",
			message: &crowparty.PartyRequest{Request: &crowparty.PartyRequest_LeaveRoom{LeaveRoom: false}},
			want: `&crowparty.PartyRequest{
	Request: &crowparty.PartyRequest_LeaveRoom{},
}
`,
		},
		{
			name:    "empty message",
			message: &gameproto.Empty{},
			want:    "&proto.Empty{}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.message.ProtoReflect().SetUnknown([]byte{0xa0, 0x06, 1})
			data, err := proto.Marshal(tt.message)
			require.NoError(t, err)
			input := filepath.Join(t.TempDir(), "input.pb")
			require.NoError(t, os.WriteFile(input, data, 0o600))
			var out bytes.Buffer
			require.NoError(t, decode(&out, string(tt.message.ProtoReflect().Descriptor().FullName()), input, "go"))
			if tt.unordered {
				require.ElementsMatch(t, strings.Split(tt.want, "\n"), strings.Split(out.String(), "\n"))
			} else {
				require.Equal(t, tt.want, out.String())
			}
			_, err = parser.ParseExpr(out.String())
			require.NoError(t, err)
		})
	}
}

func TestDecodeErrors(t *testing.T) {
	input := filepath.Join(t.TempDir(), "bad.pb")
	require.NoError(t, os.WriteFile(input, []byte{0xff}, 0o600))
	for _, tt := range []struct {
		message string
		input   string
		want    string
	}{
		{"Proto.DoesNotExist", input, "unknown message"},
		{"Proto.Empty", input + ".missing", "read input"},
		{"Proto.Empty", input, "decode Proto.Empty"},
	} {
		var out bytes.Buffer
		require.ErrorContains(t, decode(&out, tt.message, tt.input, "go"), tt.want)
		require.Empty(t, out.String())
	}
}

func TestGoLiteralSpecialValues(t *testing.T) {
	message := &descriptorpb.FileDescriptorProto{Name: proto.String("")}
	value := goLiteral(reflect.ValueOf(message), 0)
	require.Contains(t, value, `Name: func() *string { v := string(""); return &v }()`)
	_, err := parser.ParseExpr(value)
	require.NoError(t, err)

	for _, value := range []float32{float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))} {
		literal := goLiteral(reflect.ValueOf(value), 0)
		require.True(t, strings.HasPrefix(literal, "float32(math."), literal)
		_, err := parser.ParseExpr(literal)
		require.NoError(t, err)
	}
}
