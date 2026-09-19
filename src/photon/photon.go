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

	addr := net.UDPAddrFromAddrPort(netip.AddrPortFrom(cfg.Host, 5055))
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	return server.Serve(conn)
}
