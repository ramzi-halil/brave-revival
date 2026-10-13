package crow

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"maps"
	"sync"

	"example.com/brave-revival/src/proto/crowparty"
)

type partyHandler struct {
	crowparty.UnimplementedCrowPartyServer

	roomLock       sync.Mutex
	activePlayerId uint64
	room           *crowparty.Room
}

func (h *partyHandler) handleChangePlayerProperty(playerId uint64, playerProperty *crowparty.CustomProperty) *crowparty.CustomProperty {
	for _, player := range h.room.Players {
		if player.Id == playerId {
			maps.Copy(player.Property.Map, playerProperty.Map)
			return player.Property
		}
	}
	return nil
}

func (h *partyHandler) handleChangeRoomProperty(roomProperty *crowparty.CustomProperty) []*crowparty.PartyResponse {
	maps.Copy(h.room.Property.Map, roomProperty.Map)

	var extraResponses []*crowparty.PartyResponse
	if newStage, ok := roomProperty.Map["stage_id"]; ok {
		for _, player := range h.room.Players {
			extraResponses = append(extraResponses, &crowparty.PartyResponse{
				SenderPlayerId: player.Id,
				Response: &crowparty.PartyResponse_ChangePlayerProperty{
					ChangePlayerProperty: &crowparty.CustomProperty{
						Map: map[string]string{"stage_id": newStage},
					},
				},
			})
		}
	}
	return extraResponses
}

func (h *partyHandler) handleJoinRoom(req *crowparty.JoinRoom) {
	if h.room == nil {
		h.room = &crowparty.Room{
			Id: "room",
			Property: &crowparty.CustomProperty{Map: map[string]string{
				"name": "room",
			}},
			LeaderPlayerId: req.PlayerId,
		}
	}
	if req.RoomId != "" {
		h.room.Id = req.RoomId
	}
	maps.Copy(h.room.Property.Map, req.RoomProperty.Map)
	player := &crowparty.Player{
		Id:       req.PlayerId,
		Property: req.PlayerProperty,
	}
	h.room.Players = append(h.room.Players, player)
}

func (h *partyHandler) Connect(stream crowparty.CrowParty_ConnectServer) error {
	for {
		request, err := stream.Recv()
		slog.Info("CrowParty.CrowParty/Connect", "request", request, "err", err)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		h.roomLock.Lock()

		responses := make([]*crowparty.PartyResponse, 0, 1)
		appendResponse := func(sender uint64, resp crowparty.PartyResponseResponse) {
			responses = append(responses, &crowparty.PartyResponse{
				MessageId:      request.MessageId,
				SenderPlayerId: sender,
				Response:       resp,
			})
		}

		switch req := request.Request.(type) {
		case *crowparty.PartyRequest_ResumeRoom:
			h.activePlayerId = req.ResumeRoom.PlayerId
			if h.room != nil {
				h.handleChangePlayerProperty(req.ResumeRoom.PlayerId, req.ResumeRoom.PlayerProperty)
				appendResponse(h.activePlayerId, &crowparty.PartyResponse_JoinRoom{JoinRoom: h.room})
			}
		case *crowparty.PartyRequest_JoinRoom:
			h.handleJoinRoom(req.JoinRoom)
			appendResponse(req.JoinRoom.PlayerId, &crowparty.PartyResponse_JoinRoom{JoinRoom: h.room})
		case *crowparty.PartyRequest_ChangePlayerProperty:
			if h.room != nil {
				props := h.handleChangePlayerProperty(h.activePlayerId, req.ChangePlayerProperty)
				appendResponse(h.activePlayerId, &crowparty.PartyResponse_ChangePlayerProperty{ChangePlayerProperty: props})
			}
		case *crowparty.PartyRequest_ChangeRoomProperty:
			if h.room != nil {
				extraResponses := h.handleChangeRoomProperty(req.ChangeRoomProperty)
				for _, resp := range extraResponses {
					appendResponse(resp.SenderPlayerId, resp.Response)
				}
				appendResponse(h.room.LeaderPlayerId, &crowparty.PartyResponse_ChangeRoomProperty{ChangeRoomProperty: h.room.Property})
			}
		case *crowparty.PartyRequest_RaiseEvent:
			appendResponse(h.activePlayerId, &crowparty.PartyResponse_RaiseEvent{RaiseEvent: req.RaiseEvent})
		case *crowparty.PartyRequest_DisbandRoom:
			for _, player := range h.room.Players {
				appendResponse(player.Id, &crowparty.PartyResponse_LeavePlayer{LeavePlayer: player})
			}
			appendResponse(h.activePlayerId, &crowparty.PartyResponse_DisbandRoom{DisbandRoom: true})
			h.room = nil
		default:
			slog.Warn("CrowParty.CrowParty/Connect: unhandled request type", "type", request.Request)
		}

		if len(responses) == 0 {
			appendResponse(h.activePlayerId, nil)
		}

		h.roomLock.Unlock()

		for _, response := range responses {
			if err := stream.Send(response); err != nil {
				return err
			}
		}
	}
}

func (*partyHandler) IsPartyMember(context.Context, *crowparty.IsPartyMemberRequest) (*crowparty.IsPartyMemberResponse, error) {
	slog.Warn("CrowParty.CrowParty/IsPartyMember")
	// TODO: Look up the requested room membership.
	return &crowparty.IsPartyMemberResponse{}, nil
}

func (*partyHandler) SearchRooms(context.Context, *crowparty.PartySearchRequest) (*crowparty.PartySearchResponse, error) {
	slog.Debug("CrowParty.CrowParty/SearchRooms")
	return &crowparty.PartySearchResponse{}, nil
}
