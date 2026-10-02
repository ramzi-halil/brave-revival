package www

import (
	"net/http"

	"example.com/brave-revival/src/proto/proto"
	"example.com/brave-revival/src/proto/puser"
)

func equipmentForge(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentIds := r.PostForm["equipment_id"]

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		updatedEquipments := make(map[uint64]*puser.Equipment, len(equipmentIds))
		for _, idString := range equipmentIds {
			id := u64(idString)
			equipment := player.Equipment.List[id]
			equipment.Level = 30 + equipment.LimitBreak*2
			updatedEquipments[id] = equipment
		}
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Equipment:  &proto.StoredEquipment{Add: updatedEquipments},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func equipmentEnhance(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))
	loopFlag := u32(r.PostForm.Get("loop_flag"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.EquipmentEnhanceResponse {
		equipment := player.Equipment.List[equipmentId]
		if loopFlag == 1 {
			equipment.Enhancement++
		}
		return &proto.EquipmentEnhanceResponse{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Equipment: &proto.StoredEquipment{
					Add: map[uint64]*puser.Equipment{equipmentId: equipment},
				},
			},
			TryCount: 1,
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func equipmentInheritEnhance(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))
	materialId := u64(r.PostForm.Get("material_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		equipment := player.Equipment.List[equipmentId]
		material := player.Equipment.List[materialId]
		equipment.Enhancement = material.Enhancement
		material.Enhancement = 0
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Equipment: &proto.StoredEquipment{
					Add: map[uint64]*puser.Equipment{
						equipmentId: equipment,
						materialId:  material,
					},
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func equipmentOptionSkillLockOrUnlock(w http.ResponseWriter, r *http.Request, protectState uint32) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))
	index := u32(r.PostForm.Get("target_param_option_index"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		equipment := player.Equipment.List[equipmentId]
		switch index {
		case 1:
			equipment.IsParamOption1Protected = protectState
		case 2:
			equipment.IsParamOption2Protected = protectState
		case 3:
			equipment.IsParamOption3Protected = protectState
		case 4:
			equipment.IsParamOption4Protected = protectState
		}
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Equipment: &proto.StoredEquipment{
					Add: map[uint64]*puser.Equipment{equipmentId: equipment},
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func equipmentOptionSkillLock(w http.ResponseWriter, r *http.Request) {
	equipmentOptionSkillLockOrUnlock(w, r, 1)
}

func equipmentOptionSkillUnlock(w http.ResponseWriter, r *http.Request) {
	equipmentOptionSkillLockOrUnlock(w, r, 0)
}

func equipmentLimitbreak(w http.ResponseWriter, r *http.Request) {
	var req proto.LimitBreakRequest
	if !readProto(r, &req) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		equipment := player.Equipment.List[req.EquipmentId]
		equipment.LimitBreak++
		equipment.Level += 2
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Equipment: &proto.StoredEquipment{
					Add: map[uint64]*puser.Equipment{req.EquipmentId: equipment},
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func equipmentEvolutionMaterial(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		equipment := player.Equipment.List[equipmentId]
		equipment.Rarity++
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Equipment: &proto.StoredEquipment{
					Add: map[uint64]*puser.Equipment{equipmentId: equipment},
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func equipmentWeaponSkillEnhanceMaterial(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))
	slot := u32(r.PostForm.Get("slot"))
	loopFlag := u32(r.PostForm.Get("loop_flag"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.EquipmentEnhanceResponse {
		equipment := player.Equipment.List[equipmentId]
		if loopFlag == 1 {
			switch slot {
			case 1:
				equipment.WeaponSkillLevel1++
			case 2:
				equipment.WeaponSkillLevel2++
			case 3:
				equipment.WeaponSkillLevel3++
			}
		}
		return &proto.EquipmentEnhanceResponse{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Equipment: &proto.StoredEquipment{
					Add: map[uint64]*puser.Equipment{equipmentId: equipment},
				},
			},
			TryCount: 1,
		}
	})
	writeProto(w, http.StatusOK, reply)
}
