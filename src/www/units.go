package www

import (
	"fmt"

	"example.com/brave-revival/src/proto/pmaster"
)

type Unit struct {
	ID                uint32    `json:"id"`
	Name              string    `json:"name"`
	IconM             uint32    `json:"iconM"`
	IconL             uint32    `json:"iconL"`
	IconLL            uint32    `json:"iconLL"`
	IconXL            uint32    `json:"iconXL"`
	BgRgb             string    `json:"bgRgb"`
	Story             string    `json:"story,omitempty"`
	VisitorResourceID uint32    `json:"visitorResourceId,omitempty"`
	CharacterID       uint32    `json:"characterId"`
	Models            [7]uint32 `json:"models"`
}

func (h *Handler) ModelMeshNames(id uint32) []string {
	if id == 0 {
		return nil
	}
	h.masterLock.Lock()
	defer h.masterLock.Unlock()
	for _, character := range h.master.EquipmentCharacter {
		switch id {
		case character.HairModelId:
			return []string{"ha1"}
		case character.FaceModelId:
			return []string{"f"}
		case character.HeadModelId:
			return []string{"h"}
		case character.UpperBodyModelId:
			return []string{"b"}
		case character.LowerBodyModelId:
			return []string{"l"}
		case character.WeaponModelId:
			return []string{"lh", "rh"}
		case character.DeviceModelId:
			return []string{"d"}
		}
	}
	return nil
}

func (h *Handler) Units() []Unit {
	h.masterLock.Lock()
	defer h.masterLock.Unlock()
	texts := make(map[uint32]string, len(h.master.TextLang))
	for _, text := range h.master.TextLang {
		texts[text.Id] = text.Text
	}
	characters := make(map[uint32]*pmaster.EquipmentCharacter, len(h.master.EquipmentCharacter))
	for _, character := range h.master.EquipmentCharacter {
		characters[character.Id] = character
	}
	visitors := make(map[uint32]*pmaster.AgitoVisitor, len(h.master.AgitoVisitor))
	for _, visitor := range h.master.AgitoVisitor {
		visitors[visitor.EquipmentId] = visitor
	}
	units := make([]Unit, 0, len(h.master.Equipment))
	for _, equipment := range h.master.Equipment {
		name := texts[equipment.TextName]
		if name == "" {
			name = fmt.Sprintf("#%d", equipment.Id)
		}
		character := characters[equipment.EquipmentCharacterId]
		visitor := visitors[equipment.Id]
		var story string
		if visitor != nil {
			story = texts[visitor.TextDescriptionId_1]
		}
		units = append(units, Unit{
			ID: equipment.Id, Name: name,
			IconM: equipment.IconM, IconL: equipment.IconL, IconLL: equipment.IconLl, IconXL: equipment.IconXl,
			BgRgb:             equipment.BgRgb,
			Story:             story,
			VisitorResourceID: visitor.GetResourceId(),
			CharacterID:       equipment.EquipmentCharacterId,
			Models: [7]uint32{
				character.GetHairModelId(), character.GetFaceModelId(), character.GetHeadModelId(),
				character.GetUpperBodyModelId(), character.GetLowerBodyModelId(),
				character.GetWeaponModelId(), character.GetDeviceModelId(),
			},
		})
	}
	return units
}
