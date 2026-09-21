package crow

import (
	"context"
	"log/slog"
	"net"
	"net/netip"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/crownotify"
	"google.golang.org/grpc"
)

type notifyHandler struct {
	crownotify.UnimplementedCrowNotifyServer
}

func (*notifyHandler) Notify(req *crownotify.NotifyRequest, stream crownotify.CrowNotify_NotifyServer) error {
	slog.Debug("CrowNotify.CrowNotify/Notify", "req", req)
	// TODO: Stream notification events to the connected client.
	<-stream.Context().Done()
	return nil
}

func (*notifyHandler) CloseNotify(context.Context, *crownotify.CloseNotifyRequest) (*crownotify.CloseNotifyResult, error) {
	slog.Debug("CrowNotify.CrowNotify/CloseNotify")
	// TODO: Close the requested notification stream.
	return &crownotify.CloseNotifyResult{}, nil
}

func (*notifyHandler) GetAnnounce(context.Context, *crownotify.NullRequest) (*crownotify.Announce, error) {
	slog.Debug("CrowNotify.CrowNotify/GetAnnounce")
	// TODO: Return the current announcement.
	return &crownotify.Announce{}, nil
}

func (*notifyHandler) GetMaintenance(context.Context, *crownotify.NullRequest) (*crownotify.Maintenance, error) {
	slog.Debug("CrowNotify.CrowNotify/GetMaintenance")
	// TODO: Return the current maintenance window.
	return &crownotify.Maintenance{}, nil
}

func (*notifyHandler) GetNotifierCount(context.Context, *crownotify.GetNotifierCountRequest) (*crownotify.GetNotifierCountResult, error) {
	slog.Debug("CrowNotify.CrowNotify/GetNotifierCount")
	// TODO: Return the number of connected notifiers.
	return &crownotify.GetNotifierCountResult{}, nil
}

func (*notifyHandler) GetNotifierList(context.Context, *crownotify.GetNotifierListRequest) (*crownotify.GetNotifierListResult, error) {
	slog.Debug("CrowNotify.CrowNotify/GetNotifierList")
	// TODO: Return the connected notifier list.
	return &crownotify.GetNotifierListResult{}, nil
}

func (*notifyHandler) SetAnnounce(context.Context, *crownotify.AnnounceRequest) (*crownotify.NullResponse, error) {
	slog.Debug("CrowNotify.CrowNotify/SetAnnounce")
	// TODO: Store and broadcast the announcement.
	return &crownotify.NullResponse{}, nil
}

func (*notifyHandler) SetMaintenance(context.Context, *crownotify.MaintenanceRequest) (*crownotify.NullResponse, error) {
	slog.Debug("CrowNotify.CrowNotify/SetMaintenance")
	// TODO: Store and broadcast the maintenance window.
	return &crownotify.NullResponse{}, nil
}

func (*notifyHandler) SendNotify(context.Context, *crownotify.SendNotifyRequest) (*crownotify.NullResponse, error) {
	slog.Debug("CrowNotify.CrowNotify/SendNotify")
	// TODO: Send the notification to its recipients.
	return &crownotify.NullResponse{}, nil
}

func (*notifyHandler) SendTicker(context.Context, *crownotify.SendTickerRequest) (*crownotify.NullResponse, error) {
	slog.Debug("CrowNotify.CrowNotify/SendTicker")
	// TODO: Broadcast the ticker message.
	return &crownotify.NullResponse{}, nil
}

func RunNotify(cfg *config.Config) error {
	address := net.TCPAddrFromAddrPort(netip.AddrPortFrom(cfg.Host, cfg.NotifyPort))
	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		return err
	}
	defer listener.Close()

	server := grpc.NewServer()
	crownotify.RegisterCrowNotifyServer(server, &notifyHandler{})
	return server.Serve(listener)
}
