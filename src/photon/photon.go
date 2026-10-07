package photon

import (
	"log/slog"
	"net"
	"net/netip"

	"example.com/brave-revival/src/config"
)

func RunLobby(cfg *config.Config) error {
	server := NewServer()
	server.HandleOperationRequest(OperationAuthenticate, func(client *Client, request OperationRequest) {
		slog.Info("Handle authenticate", "request", request)
		client.SendOperationResponse(OperationResponse{
			Code: request.Code,
			Parameters: Parameters{
				ParameterMatchmakingType: Integer(0),
				ParameterSecret:          String("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="),
			},
		})
	})
	server.HandleOperationRequest(OperationCreateRoom, func(client *Client, request OperationRequest) {
		slog.Info("Handle create room", "request", request)
		if len(request.Parameters) == 0 {
			// 5055 room creation = load-balancing (we'll reply with the same server)
			client.SendOperationResponse(OperationResponse{
				Code: request.Code,
				Parameters: Parameters{
					ParameterAddress:  String(net.JoinHostPort(cfg.AdvertiseHost, "5055")),
					ParameterRoomName: String("room"),
				},
			})
			return
		}
		// TODO: Implement 5056 room creation.
		client.Disconnect(DisconnectLogic)
	})
	server.HandleOperationRequest(OperationJoinRandomRoom, func(client *Client, request OperationRequest) {
		// TODO: fallback to non-join-random-room case.
		client.Disconnect(DisconnectLogic)
	})

	addr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(cfg.Host, 5055))
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	return server.Serve(conn)
}
