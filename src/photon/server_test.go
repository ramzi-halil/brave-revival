package photon

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerLifecycleAndMessages(t *testing.T) {
	serverConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	require.NoError(t, err)
	server := NewServer()
	server.ClientTimeout = time.Minute
	connected := make(chan *Client, 1)
	server.OnConnect = func(client *Client) { connected <- client }
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(serverConn) }()
	t.Cleanup(func() {
		_ = server.Close()
		require.ErrorIs(t, <-serveDone, ErrServerClosed)
	})

	clientConn, err := net.DialUDP("udp4", nil, serverConn.LocalAddr().(*net.UDPAddr))
	require.NoError(t, err)
	defer clientConn.Close()
	challenge := uint32(0x12345678)

	connectPayload := make([]byte, 32)
	binary.BigEndian.PutUint16(connectPayload[2:4], defaultMTU)
	binary.BigEndian.PutUint16(connectPayload[6:8], 32768)
	connectPayload[11] = 2
	writeTestPacket(t, clientConn, packet{
		peerID: PeerIDUnconnected, challenge: challenge,
		commands: []command{{typeCode: commandConnect, channel: 255, flags: commandFlagReliable, reserved: 4, rsn: 1, payload: connectPayload}},
	})

	first := readTestPacket(t, clientConn)
	second := readTestPacket(t, clientConn)
	ack, verify := command{}, command{}
	for _, p := range []packet{first, second} {
		for _, cmd := range p.commands {
			switch cmd.typeCode {
			case commandACK:
				ack = cmd
			case commandVerifyConnect:
				verify = cmd
			}
		}
	}
	require.Len(t, ack.payload, 8)
	require.Equal(t, uint32(1), binary.BigEndian.Uint32(ack.payload[:4]))
	require.Len(t, verify.payload, 32)
	peerID := binary.BigEndian.Uint16(verify.payload[:2])
	require.NotEqual(t, PeerIDServer, peerID)
	require.NotEqual(t, PeerIDUnconnected, peerID)
	ackServerCommand(t, clientConn, peerID, challenge, verify)

	initBody := make([]byte, 39)
	initBody[0], initBody[1] = 1, 6
	copy(initBody[7:], "LoadBalancing")
	initContent := append([]byte{243, MessageInit}, initBody...)
	writeTestPacket(t, clientConn, packet{
		peerID: peerID, challenge: challenge,
		commands: []command{{typeCode: commandReliable, channel: 0, flags: commandFlagReliable, rsn: 1, payload: initContent}},
	})
	initACK := readUntilCommand(t, clientConn, commandACK)
	require.GreaterOrEqual(t, len(initACK.payload), 4)
	require.Equal(t, uint32(1), binary.BigEndian.Uint32(initACK.payload[:4]))
	initResponse := readUntilCommand(t, clientConn, commandReliable)
	require.Equal(t, []byte{243, MessageInitResponse, 0}, initResponse.payload)
	ackServerCommand(t, clientConn, peerID, challenge, initResponse)
	var client *Client
	select {
	case client = <-connected:
	case <-time.After(time.Second):
		require.FailNow(t, "OnConnect was not called")
	}
	require.Equal(t, peerID, client.ID)
	require.Equal(t, "LoadBalancing", client.AppID)

	writeTestPacket(t, clientConn, packet{
		peerID: peerID, challenge: challenge,
		commands: []command{{typeCode: commandPing, channel: 255}},
	})
	ping := readUntilCommand(t, clientConn, commandPing)
	require.Equal(t, byte(255), ping.channel)
	require.Empty(t, ping.payload)

	responses := make(chan OperationResponse, 1)
	server.HandleOperationResponse(42, func(_ *Client, response OperationResponse) { responses <- response })
	responseBody, err := MarshalOperationResponse(OperationResponse{Code: 42, ReturnCode: -1, DebugMessage: Null{}, Parameters: Parameters{7: String("seen")}})
	require.NoError(t, err)
	writeTestPacket(t, clientConn, packet{
		peerID: peerID, challenge: challenge,
		commands: []command{{typeCode: commandReliable, channel: 0, flags: commandFlagReliable, rsn: 2, payload: append([]byte{243, MessageOperationResponse}, responseBody...)}},
	})
	_ = readUntilCommand(t, clientConn, commandACK)
	select {
	case response := <-responses:
		require.Equal(t, byte(42), response.Code)
		require.Equal(t, int16(-1), response.ReturnCode)
		require.Equal(t, String("seen"), response.Parameters[7])
	case <-time.After(time.Second):
		require.FailNow(t, "operation response handler was not called")
	}

	require.NoError(t, server.SendOperationRequest(peerID, OperationRequest{Code: 8, Parameters: Parameters{1: Integer(9)}}))
	requestCommand := readUntilCommand(t, clientConn, commandReliable)
	require.GreaterOrEqual(t, len(requestCommand.payload), 2)
	require.Equal(t, byte(MessageOperation), requestCommand.payload[1])
	request, err := UnmarshalOperationRequest(requestCommand.payload[2:])
	require.NoError(t, err)
	require.Equal(t, byte(8), request.Code)
	require.Equal(t, Integer(9), request.Parameters[1])
	ackServerCommand(t, clientConn, peerID, challenge, requestCommand)

	require.NoError(t, client.SendEvent(Event{Code: 3, Parameters: Parameters{2: Boolean(true)}}))
	eventCommand := readUntilCommand(t, clientConn, commandReliable)
	require.GreaterOrEqual(t, len(eventCommand.payload), 2)
	require.Equal(t, byte(MessageEvent), eventCommand.payload[1])
	event, err := UnmarshalEvent(eventCommand.payload[2:])
	require.NoError(t, err)
	require.Equal(t, byte(3), event.Code)
	require.Equal(t, Boolean(true), event.Parameters[2])
	ackServerCommand(t, clientConn, peerID, challenge, eventCommand)

	largeData := bytes.Repeat([]byte{0xa5}, 2500)
	require.NoError(t, client.SendEvent(Event{Code: 4, Parameters: Parameters{5: ByteArray(largeData)}}))
	firstFragment := readUntilCommand(t, clientConn, commandFragment)
	fragmentCount := binary.BigEndian.Uint32(firstFragment.payload[4:8])
	fragments := []command{firstFragment}
	for uint32(len(fragments)) < fragmentCount {
		fragments = append(fragments, readUntilCommand(t, clientConn, commandFragment))
	}
	total := binary.BigEndian.Uint32(firstFragment.payload[12:16])
	assembled := make([]byte, total)
	for _, fragment := range fragments {
		offset := binary.BigEndian.Uint32(fragment.payload[16:20])
		copy(assembled[offset:], fragment.payload[20:])
		ackServerCommand(t, clientConn, peerID, challenge, fragment)
	}
	require.GreaterOrEqual(t, len(assembled), 2)
	require.Equal(t, byte(MessageEvent), assembled[1])
	largeEvent, err := UnmarshalEvent(assembled[2:])
	require.NoError(t, err)
	require.Equal(t, ByteArray(largeData), largeEvent.Parameters[5])
}

func writeTestPacket(t *testing.T, conn *net.UDPConn, p packet) {
	t.Helper()
	data, err := marshalPacket(p)
	require.NoError(t, err)
	_, err = conn.Write(data)
	require.NoError(t, err)
}

func readTestPacket(t *testing.T, conn *net.UDPConn) packet {
	t.Helper()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
	data := make([]byte, 65535)
	n, err := conn.Read(data)
	require.NoError(t, err)
	p, err := parsePacket(data[:n])
	require.NoError(t, err)
	return p
}

func readUntilCommand(t *testing.T, conn *net.UDPConn, commandType byte) command {
	t.Helper()
	for range 10 {
		p := readTestPacket(t, conn)
		for _, cmd := range p.commands {
			if cmd.typeCode == commandType {
				return cmd
			}
		}
	}
	require.FailNowf(t, "command not received", "type %d", commandType)
	return command{}
}

func ackServerCommand(t *testing.T, conn *net.UDPConn, peerID uint16, challenge uint32, cmd command) {
	t.Helper()
	payload := make([]byte, 8)
	binary.BigEndian.PutUint32(payload[:4], cmd.rsn)
	writeTestPacket(t, conn, packet{
		peerID: peerID, challenge: challenge,
		commands: []command{{typeCode: commandACK, channel: cmd.channel, payload: payload}},
	})
}
