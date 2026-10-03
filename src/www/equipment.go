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

func attachRune(runeSlot *uint64, changedRunes map[uint64]*puser.Rune, runes *proto.StoredRune, runeId uint64) {
	if runeId == 0 {
		return
	}
	*runeSlot = runeId
	r := runes.List[runeId]
	r.IsEquipped = 1
	changedRunes[runeId] = r
}

func detachRune(runeSlot *uint64, changedRunes map[uint64]*puser.Rune, runes *proto.StoredRune, shouldDetach bool) {
	runeId := *runeSlot
	if !shouldDetach || runeId == 0 {
		return
	}
	*runeSlot = 0
	r := runes.List[runeId]
	r.IsEquipped = 0
	changedRunes[runeId] = r
}

func equipmentRuneAttach(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))
	rune1 := u64(r.PostForm.Get("rune1"))
	rune2 := u64(r.PostForm.Get("rune2"))
	rune3 := u64(r.PostForm.Get("rune3"))
	rune4 := u64(r.PostForm.Get("rune4"))
	rune5 := u64(r.PostForm.Get("rune5"))
	rune6 := u64(r.PostForm.Get("rune6"))
	rune7 := u64(r.PostForm.Get("rune7"))
	rune8 := u64(r.PostForm.Get("rune8"))
	rune9 := u64(r.PostForm.Get("rune9"))
	rune10 := u64(r.PostForm.Get("rune10"))
	rune11 := u64(r.PostForm.Get("rune11"))
	rune12 := u64(r.PostForm.Get("rune12"))
	runeEx := u64(r.PostForm.Get("rune_ex"))
	runeEx2 := u64(r.PostForm.Get("rune_ex2"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		equipment := player.Equipment.List[equipmentId]
		changedRunes := make(map[uint64]*puser.Rune)
		attachRune(&equipment.Rune1, changedRunes, player.Rune, rune1)
		attachRune(&equipment.Rune2, changedRunes, player.Rune, rune2)
		attachRune(&equipment.Rune3, changedRunes, player.Rune, rune3)
		attachRune(&equipment.Rune4, changedRunes, player.Rune, rune4)
		attachRune(&equipment.Rune5, changedRunes, player.Rune, rune5)
		attachRune(&equipment.Rune6, changedRunes, player.Rune, rune6)
		attachRune(&equipment.Rune7, changedRunes, player.Rune, rune7)
		attachRune(&equipment.Rune8, changedRunes, player.Rune, rune8)
		attachRune(&equipment.Rune9, changedRunes, player.Rune, rune9)
		attachRune(&equipment.Rune10, changedRunes, player.Rune, rune10)
		attachRune(&equipment.Rune11, changedRunes, player.Rune, rune11)
		attachRune(&equipment.Rune12, changedRunes, player.Rune, rune12)
		attachRune(&equipment.RuneEx, changedRunes, player.Rune, runeEx)
		attachRune(&equipment.RuneEx2, changedRunes, player.Rune, runeEx2)
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Equipment: &proto.StoredEquipment{
					Add: map[uint64]*puser.Equipment{equipmentId: equipment},
				},
				Rune: &proto.StoredRune{
					Add: changedRunes,
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func equipmentRuneDetach(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))
	rune1 := r.PostForm.Get("rune1") == "True"
	rune2 := r.PostForm.Get("rune2") == "True"
	rune3 := r.PostForm.Get("rune3") == "True"
	rune4 := r.PostForm.Get("rune4") == "True"
	rune5 := r.PostForm.Get("rune5") == "True"
	rune6 := r.PostForm.Get("rune6") == "True"
	rune7 := r.PostForm.Get("rune7") == "True"
	rune8 := r.PostForm.Get("rune8") == "True"
	rune9 := r.PostForm.Get("rune9") == "True"
	rune10 := r.PostForm.Get("rune10") == "True"
	rune11 := r.PostForm.Get("rune11") == "True"
	rune12 := r.PostForm.Get("rune12") == "True"
	runeEx := r.PostForm.Get("rune_ex") == "True"
	runeEx2 := r.PostForm.Get("rune_ex2") == "True"

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		equipment := player.Equipment.List[equipmentId]
		changedRunes := make(map[uint64]*puser.Rune)
		detachRune(&equipment.Rune1, changedRunes, player.Rune, rune1)
		detachRune(&equipment.Rune2, changedRunes, player.Rune, rune2)
		detachRune(&equipment.Rune3, changedRunes, player.Rune, rune3)
		detachRune(&equipment.Rune4, changedRunes, player.Rune, rune4)
		detachRune(&equipment.Rune5, changedRunes, player.Rune, rune5)
		detachRune(&equipment.Rune6, changedRunes, player.Rune, rune6)
		detachRune(&equipment.Rune7, changedRunes, player.Rune, rune7)
		detachRune(&equipment.Rune8, changedRunes, player.Rune, rune8)
		detachRune(&equipment.Rune9, changedRunes, player.Rune, rune9)
		detachRune(&equipment.Rune10, changedRunes, player.Rune, rune10)
		detachRune(&equipment.Rune11, changedRunes, player.Rune, rune11)
		detachRune(&equipment.Rune12, changedRunes, player.Rune, rune12)
		detachRune(&equipment.RuneEx, changedRunes, player.Rune, runeEx)
		detachRune(&equipment.RuneEx2, changedRunes, player.Rune, runeEx2)
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Equipment: &proto.StoredEquipment{
					Add: map[uint64]*puser.Equipment{equipmentId: equipment},
				},
				Rune: &proto.StoredRune{
					Add: changedRunes,
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func equipmentRuneDetachAll(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		equipment := player.Equipment.List[equipmentId]
		changedRunes := make(map[uint64]*puser.Rune)
		detachRune(&equipment.Rune1, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune2, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune3, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune4, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune5, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune6, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune7, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune8, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune9, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune10, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune11, changedRunes, player.Rune, true)
		detachRune(&equipment.Rune12, changedRunes, player.Rune, true)
		detachRune(&equipment.RuneEx, changedRunes, player.Rune, true)
		detachRune(&equipment.RuneEx2, changedRunes, player.Rune, true)
		return &proto.Nocontent{
			StoredData: &proto.StoredData{
				Generation: player.Generation,
				Player:     player.Player,
				Equipment: &proto.StoredEquipment{
					Add: map[uint64]*puser.Equipment{equipmentId: equipment},
				},
				Rune: &proto.StoredRune{
					Add: changedRunes,
				},
			},
		}
	})
	writeProto(w, http.StatusOK, reply)
}

func equipmentProtectLockOrUnlock(w http.ResponseWriter, r *http.Request, protectState uint32) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		equipment := player.Equipment.List[equipmentId]
		equipment.IsProtected = protectState
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

func equipmentProtectLock(w http.ResponseWriter, r *http.Request) {
	equipmentProtectLockOrUnlock(w, r, 1)
}

func equipmentProtectUnlock(w http.ResponseWriter, r *http.Request) {
	equipmentProtectLockOrUnlock(w, r, 0)
}

func equipmentAwakening(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		equipment := player.Equipment.List[equipmentId]
		equipment.Awakening++
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

func equipmentAwakeningreset(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	equipmentId := u64(r.PostForm.Get("equipment_id"))

	reply := getHandler(r).writePlayer(func(player *proto.StoredData) *proto.Nocontent {
		equipment := player.Equipment.List[equipmentId]
		equipment.Awakening = 0
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
