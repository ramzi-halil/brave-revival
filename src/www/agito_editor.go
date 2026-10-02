package www

import (
	"errors"
	"fmt"
	"os"

	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
	"google.golang.org/protobuf/encoding/protojson"
	pb "google.golang.org/protobuf/proto"
)

var ErrInvalidAgitoSelection = errors.New("invalid Agito selection")

type AgitoSelection struct {
	AreaID    uint32   `json:"areaId"`
	ItemID    uint32   `json:"itemId"`
	LineupIDs []uint32 `json:"lineupIds"`
}

type AgitoEditorArea struct {
	AgitoSelection
	Type uint32 `json:"type"`
}

type AgitoEditorChoice struct {
	ID         uint32 `json:"id"`
	Name       string `json:"name"`
	ResourceID uint32 `json:"resourceId"`
}

type AgitoEditorItem struct {
	AgitoEditorChoice
	Type      uint32    `json:"type"`
	LineupIDs [3]uint32 `json:"lineupIds"`
}

type AgitoEditorLineup struct {
	AgitoEditorChoice
	LineupID      uint32 `json:"lineupId"`
	EquipmentID   uint32 `json:"equipmentId"`
	MemberID      uint32 `json:"memberId"`
	MemberName    string `json:"memberName"`
	MotionPattern uint32 `json:"motionPattern"`
}

type AgitoEditorData struct {
	Areas   []AgitoEditorArea   `json:"areas"`
	Items   []AgitoEditorItem   `json:"items"`
	Lineups []AgitoEditorLineup `json:"lineups"`
}

func (h *Handler) AgitoEditor() AgitoEditorData {
	h.masterLock.Lock()
	defer h.masterLock.Unlock()
	h.playerLock.Lock()
	defer h.playerLock.Unlock()
	data := AgitoEditorData{
		Areas: []AgitoEditorArea{}, Items: []AgitoEditorItem{}, Lineups: []AgitoEditorLineup{},
	}
	texts := make(map[uint32]string, len(h.master.TextLang))
	for _, text := range h.master.TextLang {
		texts[text.Id] = text.Text
	}
	name := func(textID, id uint32) string {
		if text := texts[textID]; text != "" {
			return text
		}
		return fmt.Sprintf("#%d", id)
	}
	items := make(map[uint32]AgitoEditorChoice, len(h.master.Item))
	for _, item := range h.master.Item {
		items[item.Id] = AgitoEditorChoice{ID: item.Id, Name: name(item.TextName, item.Id), ResourceID: item.ResourceId}
	}
	for _, item := range h.master.AgitoItem {
		choice, ok := items[item.ItemId]
		if !ok {
			choice = AgitoEditorChoice{ID: item.ItemId, Name: name(0, item.ItemId)}
		}
		data.Items = append(data.Items, AgitoEditorItem{
			AgitoEditorChoice: choice, Type: item.Type,
			LineupIDs: [3]uint32{item.AgitoVisitorLineupId_1, item.AgitoVisitorLineupId_2, item.AgitoVisitorLineupId_3},
		})
	}
	members := make(map[uint32]string, len(h.master.Member))
	for _, member := range h.master.Member {
		members[member.Id] = name(member.TextName, member.Id)
	}
	equipment := make(map[uint32]AgitoEditorLineup, len(h.master.Equipment))
	for _, item := range h.master.Equipment {
		memberName := members[item.MemberId]
		if memberName == "" {
			memberName = name(0, item.MemberId)
		}
		equipment[item.Id] = AgitoEditorLineup{
			AgitoEditorChoice: AgitoEditorChoice{Name: name(item.TextName, item.Id), ResourceID: item.IconM},
			MemberID:          item.MemberId, MemberName: memberName,
		}
	}
	for _, lineup := range h.master.AgitoVisitorLineup {
		if lineup.EquipmentId == 0 {
			continue
		}
		choice, ok := equipment[lineup.EquipmentId]
		if !ok {
			choice.Name = name(0, lineup.EquipmentId)
			choice.MemberName = name(0, 0)
		}
		choice.ID = lineup.Id
		choice.LineupID = lineup.LineupId
		choice.EquipmentID = lineup.EquipmentId
		choice.MotionPattern = lineup.MotionPattern
		data.Lineups = append(data.Lineups, choice)
	}
	for _, area := range h.master.AgitoItemArea {
		stored := h.player.GetAgitoItemArea().GetList()[area.Id|0x10000]
		data.Areas = append(data.Areas, AgitoEditorArea{
			Type: area.AgitoItemType,
			AgitoSelection: AgitoSelection{
				AreaID: area.Id, ItemID: stored.GetItemId(),
				LineupIDs: []uint32{stored.GetAgitoVisitorLineupId_1(), stored.GetAgitoVisitorLineupId_2(), stored.GetAgitoVisitorLineupId_3()},
			},
		})
	}
	return data
}

func (h *Handler) SaveAgitoEditor(selections []AgitoSelection) error {
	h.masterLock.Lock()
	defer h.masterLock.Unlock()
	areas := make(map[uint32]bool, len(h.master.AgitoItemArea))
	for _, area := range h.master.AgitoItemArea {
		areas[area.Id] = true
	}
	items := make(map[uint32][3]uint32, len(h.master.AgitoItem))
	for _, item := range h.master.AgitoItem {
		items[item.ItemId] = [3]uint32{item.AgitoVisitorLineupId_1, item.AgitoVisitorLineupId_2, item.AgitoVisitorLineupId_3}
	}
	lineups := make(map[uint32]bool, len(h.master.AgitoVisitorLineup))
	for _, lineup := range h.master.AgitoVisitorLineup {
		lineups[lineup.Id] = lineup.EquipmentId != 0
	}
	seen := make(map[uint32]bool, len(selections))
	for _, selection := range selections {
		if !areas[selection.AreaID] || seen[selection.AreaID] {
			return fmt.Errorf("%w: unknown or repeated area %d", ErrInvalidAgitoSelection, selection.AreaID)
		}
		seen[selection.AreaID] = true
		groups, ok := items[selection.ItemID]
		if selection.ItemID != 0 && !ok {
			return fmt.Errorf("%w: unknown item %d", ErrInvalidAgitoSelection, selection.ItemID)
		}
		if len(selection.LineupIDs) > 3 {
			return fmt.Errorf("%w: at most three visitors per area", ErrInvalidAgitoSelection)
		}
		for i, id := range selection.LineupIDs {
			if id != 0 && (groups[i] == 0 || !lineups[id]) {
				return fmt.Errorf("%w: invalid visitor %d in slot %d", ErrInvalidAgitoSelection, id, i+1)
			}
		}
	}
	if len(selections) == 0 {
		return nil
	}
	h.playerLock.Lock()
	defer h.playerLock.Unlock()
	player := pb.Clone(h.player).(*proto.StoredData)
	if player.AgitoItemArea == nil {
		player.AgitoItemArea = &proto.StoredAgitoItemArea{}
	}
	if player.AgitoItemArea.List == nil {
		player.AgitoItemArea.List = make(map[uint32]*puser.AgitoItemArea)
	}
	for _, selection := range selections {
		key := selection.AreaID | 0x10000
		area := player.AgitoItemArea.List[key]
		if area == nil {
			area = &puser.AgitoItemArea{
				PlayerId: player.GetPlayer().GetId(), RoomNumber: 1, AgitoItemAreaId: selection.AreaID,
				LineupReturnAt1: "0", LineupReturnAt2: "0", LineupReturnAt3: "0",
			}
			player.AgitoItemArea.List[key] = area
		}
		ids := []*uint32{&area.AgitoVisitorLineupId_1, &area.AgitoVisitorLineupId_2, &area.AgitoVisitorLineupId_3}
		flags := []*uint32{&area.ReceivedFlag1, &area.ReceivedFlag2, &area.ReceivedFlag3}
		times := []*string{&area.LineupReturnAt1, &area.LineupReturnAt2, &area.LineupReturnAt3}
		for i := range ids {
			var id uint32
			if i < len(selection.LineupIDs) {
				id = selection.LineupIDs[i]
			}
			if area.ItemId != selection.ItemID || *ids[i] != id {
				*times[i] = "0"
			}
			*flags[i] = 1
			*ids[i] = id
		}
		area.ItemId = selection.ItemID
	}
	player.Generation++
	if h.config.PlayerPath != "" {
		data, err := (protojson.MarshalOptions{Multiline: true}).Marshal(player)
		if err != nil {
			return fmt.Errorf("serialize player: %w", err)
		}
		if err := os.WriteFile(h.config.PlayerPath, data, 0o644); err != nil {
			return fmt.Errorf("save player: %w", err)
		}
	}
	h.player = player
	return nil
}
