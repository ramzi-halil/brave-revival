package www

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/proto"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

func TestAgitoPlayerRecommend(t *testing.T) {
	master := &pmaster.All{
		Equipment: []*pmaster.Equipment{
			{Id: 1, EquipmentCategory: 1, MemberId: 10},
			{Id: 2, EquipmentCategory: 2, MemberId: 10},
			{Id: 3, EquipmentCategory: 1, MemberId: 20},
			{Id: 4, EquipmentCategory: 2, MemberId: 30},
			{Id: 5, EquipmentCategory: 3, MemberId: 40},
		},
		AgitoItemArea: []*pmaster.AgitoItemArea{
			{Id: 209, AgitoItemType: 2, ExceptAgitoItemAreaId_1: 101},
			{Id: 405, AgitoItemType: 4},
			{Id: 101, AgitoItemType: 1},
			{Id: 102, AgitoItemType: 1, ExceptAgitoItemAreaId_3: 405},
		},
	}
	for _, itemType := range []uint32{1, 2, 3, 6, 7, 8} {
		master.AgitoFurniture = append(master.AgitoFurniture, &pmaster.AgitoFurniture{ItemId: 1000 + itemType, Type: itemType})
	}
	for id := uint32(301); id <= 304; id++ {
		master.AgitoItemArea = append(master.AgitoItemArea, &pmaster.AgitoItemArea{Id: id, AgitoItemType: 3})
	}
	for id := uint32(201); id <= 208; id++ {
		master.AgitoItemArea = append(master.AgitoItemArea, &pmaster.AgitoItemArea{
			Id: id, AgitoItemType: 2, ExceptAgitoItemAreaId_2: 401 + (id-201)/2,
		})
	}
	for id := uint32(401); id <= 404; id++ {
		master.AgitoItemArea = append(master.AgitoItemArea, &pmaster.AgitoItemArea{
			Id: id, AgitoItemType: 4, ExceptAgitoItemAreaId_1: 201 + (id-401)*2, ExceptAgitoItemAreaId_2: 202 + (id-401)*2,
		})
	}
	itemTypes := make(map[uint32]uint32)
	expectedLineups := make(map[uint32][3][]uint32)
	for itemType := uint32(1); itemType <= 4; itemType++ {
		for i := uint32(0); i < 4; i++ {
			item := &pmaster.AgitoItem{ItemId: itemType*100 + i, Type: itemType}
			item.AgitoVisitorLineupId_1 = item.ItemId*10 + 1
			item.AgitoVisitorLineupId_3 = item.ItemId*10 + 3
			firstIDs := []uint32{item.ItemId*100 + 1, item.ItemId*100 + 2}
			master.AgitoVisitorLineup = append(master.AgitoVisitorLineup,
				&pmaster.AgitoVisitorLineup{Id: firstIDs[0], LineupId: item.AgitoVisitorLineupId_1, MotionPattern: 1, EquipmentId: 1},
				&pmaster.AgitoVisitorLineup{Id: firstIDs[1], LineupId: item.AgitoVisitorLineupId_1, MotionPattern: 2, EquipmentId: 1, Weight: 999999},
				&pmaster.AgitoVisitorLineup{Id: item.ItemId*100 + 5, LineupId: item.AgitoVisitorLineupId_1},
				&pmaster.AgitoVisitorLineup{Id: item.ItemId*100 + 6, LineupId: item.AgitoVisitorLineupId_3},
			)
			secondIDs, thirdIDs := []uint32{0}, []uint32{0}
			if i == 2 {
				item.AgitoVisitorLineupId_2 = item.ItemId*10 + 2
				secondIDs = []uint32{item.ItemId*100 + 3}
				master.AgitoVisitorLineup = append(master.AgitoVisitorLineup, &pmaster.AgitoVisitorLineup{Id: secondIDs[0], LineupId: item.AgitoVisitorLineupId_2, MotionPattern: 1, EquipmentId: 1})
			}
			if i != 0 {
				thirdIDs = []uint32{item.ItemId*100 + 4}
				master.AgitoVisitorLineup = append(master.AgitoVisitorLineup, &pmaster.AgitoVisitorLineup{Id: thirdIDs[0], LineupId: item.AgitoVisitorLineupId_3, MotionPattern: 1, EquipmentId: 1})
			}
			expectedLineups[item.ItemId] = [3][]uint32{firstIDs, secondIDs, thirdIDs}
			master.AgitoItem = append(master.AgitoItem, item, item)
			itemTypes[item.ItemId] = itemType
		}
	}
	original := pb.Clone(master)
	handler := &Handler{master: master}
	for range 128 {
		r := httptest.NewRequest(http.MethodGet, "/agito/player/recommend", nil)
		r = r.WithContext(context.WithValue(r.Context(), handlerKey{}, handler))
		w := httptest.NewRecorder()
		agitoPlayerRecommend(w, r)
		require.Equal(t, http.StatusOK, w.Code)
		var reply proto.AgitoInfo
		require.NoError(t, pb.Unmarshal(w.Body.Bytes(), &reply))

		members := make(map[uint32]bool)
		for _, id := range []uint32{reply.FavoriteEquipmentId_1, reply.FavoriteEquipmentId_2, reply.FavoriteEquipmentId_3} {
			require.GreaterOrEqual(t, id, uint32(1))
			require.LessOrEqual(t, id, uint32(4))
			equipment := master.Equipment[id-1]
			require.False(t, members[equipment.MemberId])
			members[equipment.MemberId] = true
		}
		require.Len(t, reply.AgitoRoom, 1)
		room := reply.AgitoRoom[0]
		require.Equal(t, uint32(1), room.RoomNumber)
		furniture := room.AgitoFurnitureSetting
		require.Equal(t, reply.PlayerId, furniture.PlayerId)
		require.Equal(t, room.RoomNumber, furniture.RoomNumber)
		require.Equal(t, []uint32{1001, 1002, 1003, 1006, 1007, 1007, 1007, 1007, 1007, 1008, 1008, 1008, 1008, 1008, 1008}, []uint32{
			furniture.WallPaperItemId, furniture.FloorBoardItemId, furniture.TableSetItemId, furniture.SpecialFloorItemId,
			furniture.WallMediumItemId_1, furniture.WallMediumItemId_2, furniture.WallMediumItemId_3, furniture.WallMediumItemId_4, furniture.WallMediumItemId_5,
			furniture.WallSmallItemId_1, furniture.WallSmallItemId_2, furniture.WallSmallItemId_3, furniture.WallSmallItemId_4, furniture.WallSmallItemId_5, furniture.WallSmallItemId_6,
		})

		selected, usedItems := make(map[uint32]bool), make(map[uint32]bool)
		counts := make(map[uint32]int)
		for _, area := range room.AgitoItemArea {
			require.False(t, selected[area.AgitoItemAreaId])
			require.False(t, usedItems[area.ItemId])
			selected[area.AgitoItemAreaId] = true
			usedItems[area.ItemId] = true
			require.Equal(t, reply.PlayerId, area.PlayerId)
			require.Equal(t, room.RoomNumber, area.RoomNumber)
			lineups := expectedLineups[area.ItemId]
			available, chosen := 0, 0
			for i, id := range []uint32{area.AgitoVisitorLineupId_1, area.AgitoVisitorLineupId_2, area.AgitoVisitorLineupId_3} {
				if lineups[i][0] != 0 {
					available++
				}
				if id != 0 {
					require.Contains(t, lineups[i], id)
					chosen++
				}
			}
			require.True(t, chosen == 1 || chosen == available)
		}
		for _, area := range master.AgitoItemArea {
			if area.AgitoItemType == 1 || area.AgitoItemType == 3 {
				require.True(t, selected[area.Id])
			}
			if !selected[area.Id] {
				continue
			}
			counts[area.AgitoItemType]++
			require.False(t, selected[area.ExceptAgitoItemAreaId_1])
			require.False(t, selected[area.ExceptAgitoItemAreaId_2])
			require.False(t, selected[area.ExceptAgitoItemAreaId_3])
			for _, itemArea := range room.AgitoItemArea {
				if itemArea.AgitoItemAreaId == area.Id {
					require.Equal(t, area.AgitoItemType, itemTypes[itemArea.ItemId])
				}
			}
		}
		require.LessOrEqual(t, counts[2], 4)
		require.LessOrEqual(t, counts[4], 1)
	}
	require.True(t, pb.Equal(original, master))
}
