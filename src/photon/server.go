package photon

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"
)

type OperationRequestHandler func(*Client, OperationRequest)
type OperationResponseHandler func(*Client, OperationResponse)
type EventHandler func(*Client, Event)

type Server struct {
	MTU           uint16
	RetryInterval time.Duration
	ClientTimeout time.Duration
	InitResponse  func(appID string) Object
	OnConnect     func(*Client)
	OnDisconnect  func(*Client, DisconnectReason)

	mu               sync.RWMutex
	conn             *net.UDPConn
	clientsByID      map[uint16]*Client
	clientsByAddress map[string]*Client
	requestHandlers  map[byte]OperationRequestHandler
	responseHandlers map[byte]OperationResponseHandler
	eventHandlers    map[byte]EventHandler
	nextPeerID       uint16
	startedAt        time.Time
	done             chan struct{}
	closed           bool
	closeOnce        sync.Once
}

type Client struct {
	ID      uint16
	Address *net.UDPAddr
	AppID   string

	server    *Server
	challenge uint32
	mtu       uint16
	channels  byte

	mu          sync.Mutex
	nextSendRSN map[byte]uint32
	pending     map[pendingKey]*pendingCommand
	nextRecvRSN map[byte]uint32
	recvBuffer  map[byte]map[uint32]receivedCommand
	recvUSN     map[byte]uint32
	fragments   map[uint32]*fragmentSet
	lastSeen    time.Time
	initialized bool
	closed      bool
	done        chan struct{}
	dispatch    chan func()
	sendMu      sync.Mutex
}

type pendingKey struct {
	channel byte
	rsn     uint32
}

type pendingCommand struct {
	command command
	sentAt  time.Time
}

type receivedCommand struct {
	command  command
	sentTime uint32
}

type fragmentSet struct {
	count    uint32
	total    uint32
	received uint32
	parts    map[uint32]fragmentPart
}

type fragmentPart struct {
	offset uint32
	data   []byte
}

func NewServer() *Server {
	s := &Server{}
	s.initialize()
	return s
}

func (s *Server) initialize() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initializeLocked()
}

func (s *Server) initializeLocked() {
	if s.MTU == 0 {
		s.MTU = defaultMTU
	}
	if s.RetryInterval == 0 {
		s.RetryInterval = 500 * time.Millisecond
	}
	if s.ClientTimeout == 0 {
		s.ClientTimeout = 30 * time.Second
	}
	if s.clientsByID == nil {
		s.clientsByID = make(map[uint16]*Client)
	}
	if s.clientsByAddress == nil {
		s.clientsByAddress = make(map[string]*Client)
	}
	if s.requestHandlers == nil {
		s.requestHandlers = make(map[byte]OperationRequestHandler)
	}
	if s.responseHandlers == nil {
		s.responseHandlers = make(map[byte]OperationResponseHandler)
	}
	if s.eventHandlers == nil {
		s.eventHandlers = make(map[byte]EventHandler)
	}
	if s.nextPeerID == 0 {
		s.nextPeerID = 1
	}
	if s.done == nil {
		s.done = make(chan struct{})
	}
}

func (s *Server) HandleOperationRequest(code byte, handler OperationRequestHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initializeLocked()
	if handler == nil {
		delete(s.requestHandlers, code)
	} else {
		s.requestHandlers[code] = handler
	}
}

func (s *Server) HandleOperationResponse(code byte, handler OperationResponseHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initializeLocked()
	if handler == nil {
		delete(s.responseHandlers, code)
	} else {
		s.responseHandlers[code] = handler
	}
}

func (s *Server) HandleEvent(code byte, handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initializeLocked()
	if handler == nil {
		delete(s.eventHandlers, code)
	} else {
		s.eventHandlers[code] = handler
	}
}

func (s *Server) Serve(conn *net.UDPConn) error {
	if conn == nil {
		return errors.New("photon: nil UDP connection")
	}
	s.mu.Lock()
	s.initializeLocked()
	if s.closed {
		s.mu.Unlock()
		return ErrServerClosed
	}
	if s.conn != nil {
		s.mu.Unlock()
		return errors.New("photon: server is already serving")
	}
	s.conn = conn
	s.startedAt = time.Now()
	s.mu.Unlock()

	go s.maintainClients()
	buffer := make([]byte, 65535)
	for {
		n, address, err := conn.ReadFromUDP(buffer)
		if err != nil {
			s.mu.RLock()
			closed := s.closed
			s.mu.RUnlock()
			if closed || errors.Is(err, net.ErrClosed) {
				return ErrServerClosed
			}
			return fmt.Errorf("read Photon packet: %w", err)
		}
		if err := s.handleDatagram(buffer[:n], address); err != nil {
			slog.Error("Failed to handle Photon packet", "address", address, "error", err)
		}
	}
}

func (s *Server) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.initializeLocked()
		s.closed = true
		close(s.done)
		conn := s.conn
		clients := make([]*Client, 0, len(s.clientsByID))
		for _, client := range s.clientsByID {
			clients = append(clients, client)
		}
		s.clientsByID = make(map[uint16]*Client)
		s.clientsByAddress = make(map[string]*Client)
		s.mu.Unlock()
		for _, client := range clients {
			client.markClosed()
		}
		if conn != nil {
			closeErr = conn.Close()
		}
	})
	return closeErr
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.Close(); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (s *Server) LocalAddr() net.Addr {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.conn == nil {
		return nil
	}
	return s.conn.LocalAddr()
}

func (s *Server) Client(peerID uint16) (*Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.clientsByID[peerID]
	return client, ok
}

func (s *Server) Clients() []*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clients := make([]*Client, 0, len(s.clientsByID))
	for _, client := range s.clientsByID {
		clients = append(clients, client)
	}
	return clients
}

func (s *Server) SendOperationRequest(peerID uint16, request OperationRequest) error {
	client, ok := s.Client(peerID)
	if !ok {
		return ErrClientNotFound
	}
	return client.SendOperationRequest(request)
}

func (s *Server) SendOperationResponse(peerID uint16, response OperationResponse) error {
	client, ok := s.Client(peerID)
	if !ok {
		return ErrClientNotFound
	}
	return client.SendOperationResponse(response)
}

func (s *Server) SendEvent(peerID uint16, event Event) error {
	client, ok := s.Client(peerID)
	if !ok {
		return ErrClientNotFound
	}
	return client.SendEvent(event)
}

func (c *Client) SendOperationRequest(request OperationRequest) error {
	body, err := MarshalOperationRequest(request)
	if err != nil {
		return err
	}
	messageType := byte(MessageOperation)
	if request.Internal {
		messageType = MessageInternalOperation
	}
	return c.sendMessage(messageType, body)
}

func (c *Client) SendOperationResponse(response OperationResponse) error {
	body, err := MarshalOperationResponse(response)
	if err != nil {
		return err
	}
	messageType := byte(MessageOperationResponse)
	if response.Internal {
		messageType = MessageInternalOperationResponse
	}
	return c.sendMessage(messageType, body)
}

func (c *Client) SendEvent(event Event) error {
	body, err := MarshalEvent(event)
	if err != nil {
		return err
	}
	return c.sendMessage(MessageEvent, body)
}

func (c *Client) Disconnect(reason DisconnectReason) error {
	if reason < DisconnectLogic || reason > DisconnectUnknown {
		reason = DisconnectUnknown
	}
	err := c.sendReliableCommand(command{typeCode: commandDisconnect, channel: 255, flags: commandFlagReliable, reserved: byte(reason)})
	c.server.removeClient(c, reason)
	return err
}

func (c *Client) sendMessage(messageType byte, body []byte) error {
	content := make([]byte, 2+len(body))
	content[0] = 243
	content[1] = messageType
	copy(content[2:], body)
	return c.sendContent(content)
}

func (c *Client) sendContent(content []byte) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()

	c.mu.Lock()
	closed := c.closed
	mtu := int(c.mtu)
	c.mu.Unlock()
	if closed {
		return ErrClientClosed
	}
	if mtu == 0 {
		mtu = defaultMTU
	}
	if len(content)+packetHeaderSize+commandHeaderSize <= mtu {
		return c.sendReliableCommand(command{typeCode: commandReliable, channel: 0, flags: commandFlagReliable, payload: content})
	}
	chunkSize := mtu - packetHeaderSize - commandHeaderSize - 20
	if chunkSize <= 0 || len(content) > maxMessageSize {
		return ErrMessageTooLarge
	}
	count := (len(content) + chunkSize - 1) / chunkSize
	if uint64(count) > uint64(^uint32(0)) {
		return ErrMessageTooLarge
	}

	c.mu.Lock()
	startRSN := c.nextSendRSN[0]
	if startRSN == 0 {
		startRSN = 1
	}
	c.mu.Unlock()
	for i, offset := 0, 0; offset < len(content); i, offset = i+1, offset+chunkSize {
		end := min(offset+chunkSize, len(content))
		payload := make([]byte, 20+end-offset)
		binary.BigEndian.PutUint32(payload[0:4], startRSN)
		binary.BigEndian.PutUint32(payload[4:8], uint32(count))
		binary.BigEndian.PutUint32(payload[8:12], uint32(i))
		binary.BigEndian.PutUint32(payload[12:16], uint32(len(content)))
		binary.BigEndian.PutUint32(payload[16:20], uint32(offset))
		copy(payload[20:], content[offset:end])
		if err := c.sendReliableCommand(command{typeCode: commandFragment, channel: 0, flags: commandFlagReliable, payload: payload}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) sendReliableCommand(cmd command) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClientClosed
	}
	rsn := c.nextSendRSN[cmd.channel]
	if rsn == 0 {
		rsn = 1
	}
	cmd.rsn = rsn
	cmd.flags |= commandFlagReliable
	c.nextSendRSN[cmd.channel] = rsn + 1
	key := pendingKey{channel: cmd.channel, rsn: rsn}
	c.pending[key] = &pendingCommand{command: cmd, sentAt: time.Now()}
	c.mu.Unlock()
	if err := c.server.writeCommand(c, cmd); err != nil {
		c.mu.Lock()
		delete(c.pending, key)
		c.mu.Unlock()
		return err
	}
	return nil
}

func (s *Server) handleDatagram(data []byte, address *net.UDPAddr) error {
	p, err := parsePacket(data)
	if err != nil {
		return err
	}
	slogDebugPacket("Photon.recv<-", address, p)

	addressKey := address.String()
	s.mu.RLock()
	clientByAddress := s.clientsByAddress[addressKey]
	clientByID := s.clientsByID[p.peerID]
	s.mu.RUnlock()

	if p.peerID == PeerIDUnconnected {
		for _, cmd := range p.commands {
			if cmd.typeCode != commandConnect {
				continue
			}
			if clientByAddress != nil {
				if clientByAddress.challenge != p.challenge {
					return ErrUnexpectedChallenge
				}
				_ = s.writeCommand(clientByAddress, ackCommand(cmd.channel, cmd.rsn, p.sentTime))
				return nil
			}
			return s.acceptClient(address, p.challenge, p.sentTime, cmd)
		}
		return ErrUnexpectedPeer
	}
	if clientByID == nil || clientByID != clientByAddress {
		return ErrUnexpectedPeer
	}
	if clientByID.challenge != p.challenge {
		return ErrUnexpectedChallenge
	}
	client := clientByID
	client.mu.Lock()
	client.lastSeen = time.Now()
	client.mu.Unlock()
	for _, cmd := range p.commands {
		if cmd.flags&commandFlagReliable != 0 {
			if cmd.rsn == 0 {
				return protocolError(ErrMalformedPacket, "reliable command has sequence number zero")
			}
			if err := s.writeCommand(client, ackCommand(cmd.channel, cmd.rsn, p.sentTime)); err != nil {
				return err
			}
			ready, err := client.acceptReliable(cmd, p.sentTime)
			if err != nil {
				return err
			}
			for _, item := range ready {
				if err := s.handleCommand(client, item.command); err != nil {
					return err
				}
			}
			continue
		}
		if err := s.handleCommand(client, cmd); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) acceptClient(address *net.UDPAddr, challenge uint32, sentTime uint32, cmd command) error {
	if cmd.flags != commandFlagReliable || cmd.channel != 255 || cmd.reserved != 4 || cmd.rsn == 0 || len(cmd.payload) != 32 {
		return protocolError(ErrMalformedPacket, "invalid connect command")
	}
	requestedMTU := binary.BigEndian.Uint16(cmd.payload[2:4])
	if requestedMTU < 512 {
		requestedMTU = defaultMTU
	}
	s.mu.Lock()
	s.initializeLocked()
	peerID, err := s.allocatePeerIDLocked()
	if err != nil {
		s.mu.Unlock()
		return err
	}
	mtu := min(requestedMTU, s.MTU)
	client := &Client{
		ID:          peerID,
		Address:     cloneUDPAddr(address),
		server:      s,
		challenge:   challenge,
		mtu:         mtu,
		channels:    cmd.payload[11],
		nextSendRSN: make(map[byte]uint32),
		pending:     make(map[pendingKey]*pendingCommand),
		nextRecvRSN: map[byte]uint32{255: cmd.rsn + 1},
		recvBuffer:  make(map[byte]map[uint32]receivedCommand),
		recvUSN:     make(map[byte]uint32),
		fragments:   make(map[uint32]*fragmentSet),
		lastSeen:    time.Now(),
		done:        make(chan struct{}),
		dispatch:    make(chan func(), 64),
	}
	s.clientsByID[peerID] = client
	s.clientsByAddress[address.String()] = client
	s.mu.Unlock()
	go client.runDispatcher()

	if err := s.writeCommand(client, ackCommand(cmd.channel, cmd.rsn, sentTime)); err != nil {
		s.removeClient(client, DisconnectUnknown)
		return err
	}
	verifyPayload := append([]byte(nil), cmd.payload...)
	binary.BigEndian.PutUint16(verifyPayload[0:2], peerID)
	binary.BigEndian.PutUint16(verifyPayload[2:4], mtu)
	return client.sendReliableCommand(command{typeCode: commandVerifyConnect, channel: 255, flags: commandFlagReliable, payload: verifyPayload})
}

func (s *Server) allocatePeerIDLocked() (uint16, error) {
	for range uint32(PeerIDUnconnected - 1) {
		peerID := s.nextPeerID
		s.nextPeerID++
		if s.nextPeerID == PeerIDServer || s.nextPeerID == PeerIDUnconnected {
			s.nextPeerID = 1
		}
		if _, exists := s.clientsByID[peerID]; !exists {
			return peerID, nil
		}
	}
	return 0, errors.New("photon: no peer IDs available")
}

func (c *Client) acceptReliable(cmd command, sentTime uint32) ([]receivedCommand, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	next := c.nextRecvRSN[cmd.channel]
	if next == 0 {
		next = 1
	}
	if cmd.rsn < next {
		return nil, nil
	}
	if cmd.rsn > next {
		if cmd.rsn-next > 4096 {
			return nil, protocolError(ErrMalformedPacket, "reliable sequence gap is too large")
		}
		buffer := c.recvBuffer[cmd.channel]
		if buffer == nil {
			buffer = make(map[uint32]receivedCommand)
			c.recvBuffer[cmd.channel] = buffer
		}
		buffer[cmd.rsn] = receivedCommand{command: cmd, sentTime: sentTime}
		return nil, nil
	}
	ready := []receivedCommand{{command: cmd, sentTime: sentTime}}
	next++
	buffer := c.recvBuffer[cmd.channel]
	for {
		item, ok := buffer[next]
		if !ok {
			break
		}
		ready = append(ready, item)
		delete(buffer, next)
		next++
	}
	c.nextRecvRSN[cmd.channel] = next
	return ready, nil
}

func (s *Server) handleCommand(client *Client, cmd command) error {
	switch cmd.typeCode {
	case commandACK:
		if len(cmd.payload) < 4 {
			return protocolError(ErrMalformedPacket, "ACK payload is truncated")
		}
		rsn := binary.BigEndian.Uint32(cmd.payload[0:4])
		client.mu.Lock()
		delete(client.pending, pendingKey{channel: cmd.channel, rsn: rsn})
		client.mu.Unlock()
		return nil
	case commandDisconnect:
		reason := DisconnectReason(cmd.reserved)
		if reason < DisconnectLogic || reason > DisconnectUnknown {
			reason = DisconnectUnknown
		}
		s.removeClient(client, reason)
		return nil
	case commandPing:
		return s.writeCommand(client, command{typeCode: commandPing, channel: cmd.channel})
	case commandServerTime:
		return nil
	case commandReliable:
		return s.handleContent(client, cmd.payload)
	case commandUnreliable:
		if len(cmd.payload) < 4 {
			return protocolError(ErrMalformedPacket, "unreliable payload is truncated")
		}
		usn := binary.BigEndian.Uint32(cmd.payload[0:4])
		client.mu.Lock()
		last, exists := client.recvUSN[cmd.channel]
		if exists && usn <= last {
			client.mu.Unlock()
			return nil
		}
		client.recvUSN[cmd.channel] = usn
		client.mu.Unlock()
		return s.handleContent(client, cmd.payload[4:])
	case commandFragment:
		return s.handleFragment(client, cmd.payload)
	default:
		return protocolError(ErrMalformedPacket, fmt.Sprintf("unsupported command type %d", cmd.typeCode))
	}
}

func (s *Server) handleFragment(client *Client, payload []byte) error {
	if len(payload) < 20 {
		return protocolError(ErrMalformedPacket, "fragment payload is truncated")
	}
	start := binary.BigEndian.Uint32(payload[0:4])
	count := binary.BigEndian.Uint32(payload[4:8])
	number := binary.BigEndian.Uint32(payload[8:12])
	total := binary.BigEndian.Uint32(payload[12:16])
	offset := binary.BigEndian.Uint32(payload[16:20])
	content := payload[20:]
	if count == 0 || number >= count || total > maxMessageSize || count > total || len(content) == 0 || offset > total || uint64(offset)+uint64(len(content)) > uint64(total) {
		return protocolError(ErrMalformedPacket, "invalid fragment bounds")
	}
	client.mu.Lock()
	set := client.fragments[start]
	if set == nil {
		set = &fragmentSet{count: count, total: total, parts: make(map[uint32]fragmentPart)}
		client.fragments[start] = set
	}
	if set.count != count || set.total != total {
		client.mu.Unlock()
		return protocolError(ErrMalformedPacket, "inconsistent fragment metadata")
	}
	if _, exists := set.parts[number]; !exists {
		set.parts[number] = fragmentPart{offset: offset, data: append([]byte(nil), content...)}
		set.received++
	}
	if set.received != set.count {
		client.mu.Unlock()
		return nil
	}
	assembled := make([]byte, set.total)
	covered := make([]bool, set.total)
	for _, part := range set.parts {
		copy(assembled[part.offset:], part.data)
		for i := part.offset; i < part.offset+uint32(len(part.data)); i++ {
			if covered[i] {
				client.mu.Unlock()
				return protocolError(ErrMalformedPacket, "overlapping fragments")
			}
			covered[i] = true
		}
	}
	delete(client.fragments, start)
	client.mu.Unlock()
	for _, present := range covered {
		if !present {
			return protocolError(ErrMalformedPacket, "fragment set has gaps")
		}
	}
	return s.handleContent(client, assembled)
}

func (s *Server) handleContent(client *Client, content []byte) error {
	if len(content) < 2 || content[0] != 243 {
		return protocolError(ErrMalformedMessage, "invalid content signature")
	}
	messageType := content[1]
	body := content[2:]
	if messageType&0x80 != 0 {
		messageType &^= 0x80
		newBody, err := decryptMessage(body)
		if err != nil {
			return err
		}
		body = newBody
	}

	switch messageType {
	case MessageInit:
		return s.handleInit(client, body)
	case MessageOperation, MessageInternalOperation:
		request, err := UnmarshalOperationRequest(body)
		if err != nil {
			return err
		}
		request.Internal = messageType == MessageInternalOperation
		if request.Internal && request.Code == OperationInitEncryption {
			return client.SendOperationResponse(OperationResponse{
				Code: request.Code, Internal: true, DebugMessage: Null{},
				Parameters: Parameters{1: ByteArray{1}},
			})
		}

		s.mu.RLock()
		handler := s.requestHandlers[request.Code]
		s.mu.RUnlock()
		if handler != nil {
			client.enqueue(func() { handler(client, request) })
		} else {
			slog.Warn("no handler for operation request", "code", request.Code, "internal", request.Internal)
		}
		return nil
	case MessageOperationResponse, MessageInternalOperationResponse:
		response, err := UnmarshalOperationResponse(body)
		if err != nil {
			return err
		}
		response.Internal = messageType == MessageInternalOperationResponse
		s.mu.RLock()
		handler := s.responseHandlers[response.Code]
		s.mu.RUnlock()
		if handler != nil {
			client.enqueue(func() { handler(client, response) })
		} else {
			slog.Warn("no handler for operation response", "code", response.Code, "internal", response.Internal)
		}
		return nil
	case MessageEvent:
		event, err := UnmarshalEvent(body)
		if err != nil {
			return err
		}
		s.mu.RLock()
		handler := s.eventHandlers[event.Code]
		s.mu.RUnlock()
		if handler != nil {
			client.enqueue(func() { handler(client, event) })
		} else {
			slog.Warn("no handler for event", "code", event.Code)
		}
		return nil
	default:
		return protocolError(ErrMalformedMessage, fmt.Sprintf("unsupported message type %d", messageType))
	}
}

func (s *Server) handleInit(client *Client, body []byte) error {
	if len(body) != 39 || body[0] != 1 || body[1] != 6 {
		return protocolError(ErrMalformedMessage, "invalid init message")
	}
	appID := strings.TrimRight(string(body[7:39]), "\x00")
	client.mu.Lock()
	if client.initialized {
		client.mu.Unlock()
		return nil
	}
	client.AppID = appID
	client.initialized = true
	client.mu.Unlock()

	response := []byte{243, MessageInitResponse, 0}
	var responseObject Object
	if s.InitResponse != nil {
		responseObject = s.InitResponse(appID)
	} else if appID == "1" {
		responseObject = String("ResponseObject")
	}
	if responseObject != nil {
		encoded, err := MarshalObject(responseObject)
		if err != nil {
			return fmt.Errorf("marshal init response: %w", err)
		}
		response = append(response, encoded...)
	}
	if err := client.sendContent(response); err != nil {
		return err
	}
	if s.OnConnect != nil {
		client.enqueue(func() { s.OnConnect(client) })
	}
	return nil
}

func decryptMessage(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, protocolError(ErrMalformedMessage, "encrypted message has invalid length")
	}
	key := sha256.Sum256([]byte{1})
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	plain := append([]byte(nil), data...)
	cipher.NewCBCDecrypter(block, make([]byte, aes.BlockSize)).CryptBlocks(plain, plain)
	padding := int(plain[len(plain)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(plain) {
		return nil, protocolError(ErrMalformedMessage, "encrypted message has invalid padding")
	}
	for _, b := range plain[len(plain)-padding:] {
		if int(b) != padding {
			return nil, protocolError(ErrMalformedMessage, "encrypted message has invalid padding")
		}
	}
	return plain[:len(plain)-padding], nil
}

func (s *Server) writeCommand(client *Client, cmd command) error {
	s.mu.RLock()
	conn := s.conn
	startedAt := s.startedAt
	closed := s.closed
	s.mu.RUnlock()
	if closed {
		return ErrServerClosed
	}
	if conn == nil {
		return ErrServerNotStarted
	}
	p := packet{
		peerID:    PeerIDServer,
		sentTime:  uint32(time.Since(startedAt) / time.Millisecond),
		challenge: client.challenge,
		commands:  []command{cmd},
	}
	data, err := marshalPacket(p)
	if err != nil {
		return err
	}
	if len(data) > int(client.mtu) {
		return ErrMessageTooLarge
	}
	slogDebugPacket("Photon.send->", client.Address, p)
	_, err = conn.WriteToUDP(data, client.Address)
	if err != nil {
		return fmt.Errorf("write Photon packet: %w", err)
	}
	return nil
}

func (s *Server) maintainClients() {
	s.mu.RLock()
	interval := s.RetryInterval
	s.mu.RUnlock()
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(interval / 2)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case now := <-ticker.C:
			s.maintainAt(now)
		}
	}
}

func (s *Server) maintainAt(now time.Time) {
	clients := s.Clients()
	for _, client := range clients {
		client.mu.Lock()
		if now.Sub(client.lastSeen) >= s.ClientTimeout {
			client.mu.Unlock()
			s.removeClient(client, DisconnectTimeout)
			continue
		}
		var resend []command
		for _, pending := range client.pending {
			if now.Sub(pending.sentAt) >= s.RetryInterval {
				pending.sentAt = now
				resend = append(resend, pending.command)
			}
		}
		client.mu.Unlock()
		for _, cmd := range resend {
			if err := s.writeCommand(client, cmd); err != nil && !errors.Is(err, ErrClientClosed) && !errors.Is(err, ErrServerClosed) {
				slog.Error("Failed to resend Photon command", "peer_id", client.ID, "error", err)
			}
		}
	}
}

func (s *Server) removeClient(client *Client, reason DisconnectReason) {
	s.mu.Lock()
	current := s.clientsByID[client.ID]
	if current != client {
		s.mu.Unlock()
		return
	}
	delete(s.clientsByID, client.ID)
	delete(s.clientsByAddress, client.Address.String())
	s.mu.Unlock()
	client.markClosed()
	if s.OnDisconnect != nil {
		s.OnDisconnect(client, reason)
	}
}

func (c *Client) markClosed() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	close(c.done)
	c.mu.Unlock()
}

func (c *Client) enqueue(fn func()) {
	select {
	case c.dispatch <- fn:
	case <-c.done:
	}
}

func (c *Client) runDispatcher() {
	for {
		select {
		case fn := <-c.dispatch:
			func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						slog.Error("Photon handler panicked", "peer_id", c.ID, "panic", recovered)
					}
				}()
				fn()
			}()
		case <-c.done:
			return
		}
	}
}
func cloneUDPAddr(address *net.UDPAddr) *net.UDPAddr {
	clone := *address
	clone.IP = append(net.IP(nil), address.IP...)
	return &clone
}

func slogDebugPacket(msg string, address *net.UDPAddr, p packet) {
	for _, cmd := range p.commands {
		switch cmd.typeCode {
		case commandACK, commandPing:
			continue
		default:
			slog.Debug(msg, "address", address, "packet", p)
			return
		}
	}
}
