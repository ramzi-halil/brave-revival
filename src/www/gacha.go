package www

import (
	"math/rand/v2"
	"net/http"
	"strconv"

	"example.com/brave-revival/src/proto/pmaster"
	"example.com/brave-revival/src/proto/proto"
)

func gachaPurchase(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	gachaId := u32(r.PostForm.Get("id"))
	count, _ := strconv.Atoi(r.PostForm.Get("count"))
	mode := r.PostForm.Get("mode")

	handler := getHandler(r)
	handler.masterLock.Lock()
	var pool []*pmaster.GachaPool
	for _, option := range handler.master.GachaOption {
		if option.Id == gachaId {
			switch mode {
			case "2", "6":
				count *= int(option.LumpNum)
			}
			for _, entry := range handler.master.GachaPool {
				if entry.Id == option.GachaPoolId {
					pool = append(pool, entry)
				}
			}
			break
		}
	}
	inventory := make([]*proto.RewardInfo, count)
	for i := range inventory {
		entry := pool[rand.IntN(len(pool))]
		inventory[i] = &proto.RewardInfo{
			TargetType:      2,
			Quantity:        1,
			TargetId:        entry.EquipmentId,
			EquipmentRarity: entry.Rarity,
			EquipmentLevel:  entry.EquipmentLv,
			Enhancement:     entry.EquipmentEnhancement,
			LimitBreak:      entry.EquipmentLimitBreak,
		}
	}
	handler.masterLock.Unlock()

	writeProto(w, http.StatusOK, &proto.GachaResult{
		GachaId:   gachaId,
		Inventory: inventory,
	})
}
