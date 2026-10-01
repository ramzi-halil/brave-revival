package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
)

func shopBuy(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	shopItemId := u32(r.PostForm.Get("shop_item_id"))
	reply := getHandler(r).readPlayer(func(player *proto.StoredData) *proto.ShopItemReceiveList {
		return &proto.ShopItemReceiveList{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
			},
			ShopItemID: shopItemId,
		}
	})
	writeProto(w, http.StatusOK, reply)
}
