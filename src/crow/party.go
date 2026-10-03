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

func (*partyHandler) Connect(stream crowparty.CrowParty_ConnectServer) error {
	for {
		request, err := stream.Recv()
		slog.Debug("CrowParty.CrowParty/Connect", "request", request, "err", err)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		// TODO: Implement party state and return the appropriate response.
		if err := stream.Send(&crowparty.PartyResponse{MessageId: request.MessageId, SenderPlayerId: 100}); err != nil {
			return err
		}
	}
}

func (*partyHandler) IsPartyMember(context.Context, *crowparty.IsPartyMemberRequest) (*crowparty.IsPartyMemberResponse, error) {
	slog.Debug("CrowParty.CrowParty/IsPartyMember")
	// TODO: Look up the requested room membership.
	return &crowparty.IsPartyMemberResponse{}, nil
}

func (*partyHandler) SearchRooms(context.Context, *crowparty.PartySearchRequest) (*crowparty.PartySearchResponse, error) {
	slog.Debug("CrowParty.CrowParty/SearchRooms")
	// TODO: Search the available party rooms.
	return &crowparty.PartySearchResponse{}, nil
}
