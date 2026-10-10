package crow

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"example.com/brave-revival/src/proto/crowparty"
)

type partyHandler struct {
	crowparty.UnimplementedCrowPartyServer
}

func (h *partyHandler) Connect(stream crowparty.CrowParty_ConnectServer) error {
	for {
		request, err := stream.Recv()
		slog.Debug("CrowParty.CrowParty/Connect", "request", request, "err", err)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		response := &crowparty.PartyResponse{MessageId: request.MessageId}
		switch req := request.Request.(type) {
		case *crowparty.PartyRequest_ResumeRoom:
			response.SenderPlayerId = req.ResumeRoom.PlayerId
		case *crowparty.PartyRequest_JoinRoom:
			response.SenderPlayerId = req.JoinRoom.PlayerId
			response.Response = &crowparty.PartyResponse_JoinRoom{
				JoinRoom: &crowparty.Room{
					Id:             "room",
					Property:       req.JoinRoom.RoomProperty,
					LeaderPlayerId: req.JoinRoom.PlayerId,
					Players: []*crowparty.Player{{
						Id:       req.JoinRoom.PlayerId,
						Property: req.JoinRoom.PlayerProperty,
					}},
				},
			}
		default:
			slog.Warn("CrowParty.CrowParty/Connect: unhandled request type", "type", request.Request)
		}

		// TODO: Implement party state and return the appropriate response.
		if err := stream.Send(response); err != nil {
			return err
		}
	}
}

func (*partyHandler) IsPartyMember(context.Context, *crowparty.IsPartyMemberRequest) (*crowparty.IsPartyMemberResponse, error) {
	slog.Warn("CrowParty.CrowParty/IsPartyMember")
	// TODO: Look up the requested room membership.
	return &crowparty.IsPartyMemberResponse{}, nil
}

func (*partyHandler) SearchRooms(context.Context, *crowparty.PartySearchRequest) (*crowparty.PartySearchResponse, error) {
	slog.Warn("CrowParty.CrowParty/SearchRooms")
	// TODO: Search the available party rooms.
	return &crowparty.PartySearchResponse{}, nil
}
