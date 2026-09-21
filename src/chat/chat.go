package chat

import (
	"context"
	"log/slog"
	"net/http"
	"net/netip"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proto/prealtime"
	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"
)

func handle(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.Warn("Failed to accept chat WebSocket connection", "err", err)
		return
	}
	defer conn.CloseNow()

	ctx := context.Background()
	for {
		// messages from client has this format: (seq_num:u32, op_code:u8, contents:[u8])
		// messages from server has this format: (res:u8, seq_num:u32, op_code:u8, contents:[u8])
		// we always reply (1, seq_num, 0, []),
		// except for op_code=5 (StatNumSub) which we have to reply a concrete result.
		messageType, message, err := conn.Read(ctx)
		slog.Debug("Chat message", "message", message, "err", err)
		if err != nil {
			status := websocket.CloseStatus(err)
			if status != websocket.StatusNormalClosure && status != websocket.StatusGoingAway {
				slog.Warn("Failed to read chat WebSocket message", "err", err)
			}
			return
		}

		if len(message) < 5 {
			slog.Warn("Chat WebSocket message too short", "message", message)
			continue
		}

		seqNum := message[0:4]
		opCode := message[4]

		reply := make([]byte, 6)
		reply[0] = 1
		copy(reply[1:5], seqNum)

		if opCode == 5 {
			reply[5] = 5
			var channel prealtime.Channel
			if err := proto.Unmarshal(message[5:], &channel); err != nil {
				slog.Warn("Failed to decode StatNumSub channels", "err", err)
				continue
			}

			nums := make(map[string]int32, len(channel.Channels))
			for _, w := range channel.Channels {
				nums[w] = 1
			}
			statNumSub, err := proto.Marshal(&prealtime.StatNumSub{Nums: nums})
			if err != nil {
				slog.Warn("Failed to encode StatNumSub response", "err", err)
				continue
			}
			reply = append(reply, statNumSub...)
		} else {
			reply[5] = 0
		}

		if err := conn.Write(ctx, messageType, reply); err != nil {
			slog.Warn("Failed to write chat WebSocket message", "err", err)
			return
		}
	}
}

func Run(cfg *config.Config) error {
	server := http.Server{
		Addr:    netip.AddrPortFrom(cfg.Host, cfg.ChatPort).String(),
		Handler: http.HandlerFunc(handle),
	}
	return server.ListenAndServe()
}
