package www

import (
	"testing"

	"example.com/brave-revival/src/proto/pmaster"
	"github.com/stretchr/testify/require"
)

func TestUnits(t *testing.T) {
	h := &Handler{master: &pmaster.All{
		TextLang: []*pmaster.TextLang{{Id: 9, Text: "[調マ]エレン"}, {Id: 15, Text: "Story text\nSecond line"}},
		Equipment: []*pmaster.Equipment{
			{Id: 10001010, TextName: 9, IconM: 740500101, IconL: 740600101, IconLl: 740700101, IconXl: 740800101, BgRgb: "PRISM", EquipmentCharacterId: 10},
			{Id: 20, EquipmentCharacterId: 99},
		},
		EquipmentCharacter: []*pmaster.EquipmentCharacter{{
			Id: 10, HairModelId: 1061001, FaceModelId: 1060001, HeadModelId: 1033000,
			UpperBodyModelId: 1034001, LowerBodyModelId: 1035001, WeaponModelId: 1001001, DeviceModelId: 1067001,
		}},
		AgitoVisitor: []*pmaster.AgitoVisitor{
			{EquipmentId: 10001010, TextDescriptionId_1: 15, ResourceId: 740900101},
			{EquipmentId: 99, TextDescriptionId_1: 15, ResourceId: 740900999},
		},
	}}
	require.Equal(t, []Unit{
		{
			ID: 10001010, Name: "[調マ]エレン", IconM: 740500101, IconL: 740600101, IconLL: 740700101, IconXL: 740800101,
			BgRgb: "PRISM",
			Story: "Story text\nSecond line", VisitorResourceID: 740900101,
			CharacterID: 10, Models: [7]uint32{1061001, 1060001, 1033000, 1034001, 1035001, 1001001, 1067001},
		},
		{ID: 20, Name: "#20", CharacterID: 99},
	}, h.Units())
	require.Empty(t, (&Handler{master: &pmaster.All{}}).Units())
	for id, names := range map[uint32][]string{
		1061001: {"ha1"}, 1060001: {"f"}, 1033000: {"h"}, 1034001: {"b"},
		1035001: {"l"}, 1001001: {"lh", "rh"}, 1067001: {"d"}, 0: nil, 99: nil,
	} {
		require.Equal(t, names, h.ModelMeshNames(id))
	}
}
