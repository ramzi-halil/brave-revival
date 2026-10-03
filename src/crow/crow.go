package crow

import (
	"net"
	"net/netip"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/crownotify"
	"example.com/brave-revival/src/proto/crowparty"
	"google.golang.org/grpc"
)

func Run(cfg *config.Config) error {
	address := net.TCPAddrFromAddrPort(netip.AddrPortFrom(cfg.Host, cfg.PartyPort))
	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	server := grpc.NewServer()
	crowparty.RegisterCrowPartyServer(server, &partyHandler{})
	crownotify.RegisterCrowNotifyServer(server, &notifyHandler{})
	return server.Serve(listener)
}
