package www

import (
	"os"
	"path/filepath"
	"testing"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

func agitoEditorHandler(t *testing.T) *Handler {
	t.Helper()
	master := &pmaster.All{
		AgitoItemArea: []*pmaster.AgitoItemArea{{Id: 101, AgitoItemType: 1}, {Id: 401, AgitoItemType: 4}},
		AgitoItem: []*pmaster.AgitoItem{
			{ItemId: 10, Type: 1, AgitoVisitorLineupId_1: 100},
			{ItemId: 20, Type: 4, AgitoVisitorLineupId_1: 200, AgitoVisitorLineupId_3: 300, ResourceId: 999},
		},
		Item:      []*pmaster.Item{{Id: 10, TextName: 1, ResourceId: 1000}, {Id: 20, TextName: 2, ResourceId: 2000}},
		Equipment: []*pmaster.Equipment{{Id: 30, TextName: 3, IconM: 3000, MemberId: 50}, {Id: 40, TextName: 4, IconM: 4000, MemberId: 60}},
		Member:    []*pmaster.Member{{Id: 50, TextName: 5}, {Id: 60, TextName: 6}},
		TextLang:  []*pmaster.TextLang{{Id: 1, Text: "Item one"}, {Id: 2, Text: "Item two"}, {Id: 3, Text: "Visitor one"}, {Id: 5, Text: "Member one"}},
		AgitoVisitorLineup: []*pmaster.AgitoVisitorLineup{
			{Id: 11, LineupId: 100, EquipmentId: 30, MotionPattern: 2},
			{Id: 21, LineupId: 200, EquipmentId: 40, MotionPattern: 3},
			{Id: 31, LineupId: 300},
		},
	}
	player := config.GenerateDefaultPlayer(master)
	player.AgitoItemArea.List[101|0x10000].ItemId = 10
	player.AgitoItemArea.List[101|0x10000].AgitoVisitorLineupId_1 = 11
	player.AgitoItemArea.List[101|0x10000].ReceivedFlag1 = 1
	player.AgitoItemArea.List[101|0x10000].LineupReturnAt1 = "123"
	player.AgitoItemArea.List[101|0x20000] = &puser.AgitoItemArea{RoomNumber: 2, ItemId: 10}
	return &Handler{config: &config.Config{PlayerPath: filepath.Join(t.TempDir(), "player.json")}, master: master, player: player}
}

func TestAgitoEditorData(t *testing.T) {
	h := agitoEditorHandler(t)
	before := pb.Clone(h.player)
	data := h.AgitoEditor()
	require.Equal(t, []AgitoEditorArea{
		{AgitoSelection: AgitoSelection{AreaID: 101, ItemID: 10, LineupIDs: []uint32{11, 0, 0}}, Type: 1},
		{AgitoSelection: AgitoSelection{AreaID: 401, LineupIDs: []uint32{0, 0, 0}}, Type: 4},
	}, data.Areas)
	require.Len(t, data.Items, 2)
	require.Equal(t, "Item two", data.Items[1].Name)
	require.Equal(t, uint32(2000), data.Items[1].ResourceID)
	require.Equal(t, [3]uint32{200, 0, 300}, data.Items[1].LineupIDs)
	require.Len(t, data.Lineups, 2)
	require.Equal(t, "Visitor one", data.Lineups[0].Name)
	require.Equal(t, uint32(3000), data.Lineups[0].ResourceID)
	require.Equal(t, uint32(100), data.Lineups[0].LineupID)
	require.Equal(t, uint32(2), data.Lineups[0].MotionPattern)
	require.Equal(t, uint32(3), data.Lineups[1].MotionPattern)
	require.Equal(t, uint32(50), data.Lineups[0].MemberID)
	require.Equal(t, "Member one", data.Lineups[0].MemberName)
	require.Equal(t, uint32(60), data.Lineups[1].MemberID)
	require.Equal(t, "#60", data.Lineups[1].MemberName)
	require.Equal(t, "#40", data.Lineups[1].Name)
	data.Areas[0].LineupIDs[0] = 999
	require.True(t, pb.Equal(before, h.player))
}

func TestSaveAgitoEditor(t *testing.T) {
	h := agitoEditorHandler(t)
	before := pb.Clone(h.player).(*proto.StoredData)
	// Different item types and visitor groups remain valid choices.
	require.NoError(t, h.SaveAgitoEditor([]AgitoSelection{{AreaID: 101, ItemID: 20, LineupIDs: []uint32{11, 0, 21}}}))
	require.Equal(t, before.Generation+1, h.player.Generation)
	area := h.player.AgitoItemArea.List[101|0x10000]
	require.Equal(t, uint32(20), area.ItemId)
	require.Equal(t, uint32(11), area.AgitoVisitorLineupId_1)
	require.Equal(t, uint32(21), area.AgitoVisitorLineupId_3)
	require.Equal(t, []uint32{1, 1, 1}, []uint32{area.ReceivedFlag1, area.ReceivedFlag2, area.ReceivedFlag3})
	require.Equal(t, "0", area.LineupReturnAt1)
	require.True(t, pb.Equal(before.AgitoAp, h.player.AgitoAp))
	require.True(t, pb.Equal(before.Player, h.player.Player))
	require.True(t, pb.Equal(before.AgitoItemArea.List[101|0x20000], h.player.AgitoItemArea.List[101|0x20000]))
	reloaded, err := loadPlayer(h.config.PlayerPath, h.master)
	require.NoError(t, err)
	require.True(t, pb.Equal(h.player, reloaded))

	area.ReceivedFlag1, area.ReceivedFlag2, area.ReceivedFlag3 = 0, 0, 0
	require.NoError(t, h.SaveAgitoEditor([]AgitoSelection{{AreaID: 101, ItemID: 20, LineupIDs: []uint32{11, 0, 21}}}))
	area = h.player.AgitoItemArea.List[101|0x10000]
	require.Equal(t, []uint32{1, 1, 1}, []uint32{area.ReceivedFlag1, area.ReceivedFlag2, area.ReceivedFlag3})

	require.NoError(t, h.SaveAgitoEditor([]AgitoSelection{{AreaID: 101, ItemID: 20}}))
	area = h.player.AgitoItemArea.List[101|0x10000]
	require.Zero(t, area.AgitoVisitorLineupId_1)
	require.Zero(t, area.AgitoVisitorLineupId_3)
	require.NoError(t, h.SaveAgitoEditor([]AgitoSelection{{AreaID: 101}}))
	require.Zero(t, h.player.AgitoItemArea.List[101|0x10000].ItemId)

	h.player.AgitoItemArea = nil
	require.NoError(t, h.SaveAgitoEditor([]AgitoSelection{{AreaID: 401, ItemID: 10}}))
	area = h.player.AgitoItemArea.List[401|0x10000]
	require.Equal(t, h.player.Player.Id, area.PlayerId)
	require.Equal(t, uint32(1), area.RoomNumber)
	require.Equal(t, uint32(401), area.AgitoItemAreaId)
	require.Equal(t, []uint32{1, 1, 1}, []uint32{area.ReceivedFlag1, area.ReceivedFlag2, area.ReceivedFlag3})
}

func TestSaveAgitoEditorRejectsInvalidSelections(t *testing.T) {
	h := agitoEditorHandler(t)
	before := pb.Clone(h.player)
	for _, selections := range [][]AgitoSelection{
		{{AreaID: 999}},
		{{AreaID: 101}, {AreaID: 101}},
		{{AreaID: 101, ItemID: 20}, {AreaID: 401, ItemID: 999}},
		{{AreaID: 101, ItemID: 10, LineupIDs: []uint32{999}}},
		{{AreaID: 101, ItemID: 10, LineupIDs: []uint32{31}}},
		{{AreaID: 101, ItemID: 10, LineupIDs: []uint32{11, 21}}},
		{{AreaID: 101, LineupIDs: []uint32{11}}},
		{{AreaID: 101, ItemID: 20, LineupIDs: []uint32{0, 0, 0, 0}}},
	} {
		require.ErrorIs(t, h.SaveAgitoEditor(selections), ErrInvalidAgitoSelection)
		require.True(t, pb.Equal(before, h.player))
	}
	_, err := os.Stat(h.config.PlayerPath)
	require.True(t, os.IsNotExist(err))
	require.NoError(t, h.SaveAgitoEditor(nil))
	require.True(t, pb.Equal(before, h.player))

	h.config.PlayerPath = t.TempDir()
	require.ErrorContains(t, h.SaveAgitoEditor([]AgitoSelection{{AreaID: 101}}), "save player")
	require.True(t, pb.Equal(before, h.player))
}
