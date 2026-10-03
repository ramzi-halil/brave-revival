package www

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
	"github.com/stretchr/testify/require"
	pb "google.golang.org/protobuf/proto"
)

func TestJobDeckEquipmentRemoveAll(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		name := "remove"
		if blocked {
			name = "reject without changing decks"
		}
		t.Run(name, func(t *testing.T) {
			changed := &puser.JobDeck{
				Id: 42, PlayerId: 42, Line1MainFront: 42, Line2MainFront: 99,
				Line1MainRear: 42, Line2MainStylishWeapon: 42, Line3Sub20Accessory2: 42,
				Line2StylishFlag: 8, Line3Sub1Front: 99, Line3MainCostume: 8, Line3StylishFlag: 2,
			}
			unchanged := &puser.JobDeck{Id: 2, Line3MainFront: 99}
			decks := map[uint64]*puser.JobDeck{42: changed, 2: unchanged}
			if blocked {
				decks[3] = &puser.JobDeck{Id: 3, Line2MainFront: 42, Line3MainFront: 42}
			}
			player := &proto.StoredData{JobDeck: &proto.StoredJobDeck{List: decks}}
			before := pb.Clone(player.JobDeck)
			h := &Handler{config: &config.Config{}, player: player}
			r := httptest.NewRequest(http.MethodPost, "/job/deck/equipment/remove/all", strings.NewReader("equipment_id=42"))
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r = r.WithContext(context.WithValue(r.Context(), handlerKey{}, h))
			w := httptest.NewRecorder()
			jobDeckEquipmentRemoveAll(w, r)
			require.Equal(t, http.StatusOK, w.Code)
			var reply proto.Nocontent
			require.NoError(t, pb.Unmarshal(w.Body.Bytes(), &reply))
			if blocked {
				require.Equal(t, int64(16002), reply.GetError().GetCode())
				require.Equal(t, int64(3), reply.GetError().GetLevel())
				require.Nil(t, reply.StoredData)
				require.True(t, pb.Equal(before, player.JobDeck))
				return
			}
			require.Nil(t, reply.Error)
			require.Len(t, reply.StoredData.JobDeck.Add, 2)
			expected := &puser.JobDeck{
				Id: 42, PlayerId: 42, Line2MainFront: 99,
				Line3Sub1Front: 99, Line3MainCostume: 8, Line3StylishFlag: 2,
			}
			require.True(t, pb.Equal(expected, changed))
			require.True(t, pb.Equal(expected, reply.StoredData.JobDeck.Add[42]))
			require.True(t, pb.Equal(before.(*proto.StoredJobDeck).List[2], unchanged))
		})
	}
}
