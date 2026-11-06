package photon

import (
	"errors"
	"fmt"
)

type MsgCode byte

const (
	MessageInit                      MsgCode = 0
	MessageInitResponse              MsgCode = 1
	MessageOperation                 MsgCode = 2
	MessageOperationResponse         MsgCode = 3
	MessageEvent                     MsgCode = 4
	MessageInternalOperation         MsgCode = 6
	MessageInternalOperationResponse MsgCode = 7
)

const (
	PeerIDServer      uint16 = 0
	PeerIDUnconnected uint16 = 0xffff
)

type DisconnectReason byte

const (
	DisconnectLogic   DisconnectReason = 1
	DisconnectTimeout DisconnectReason = 2
	DisconnectLimit   DisconnectReason = 3
	DisconnectUnknown DisconnectReason = 4
)

var (
	ErrServerClosed        = errors.New("photon: server closed")
	ErrClientClosed        = errors.New("photon: client closed")
	ErrClientNotFound      = errors.New("photon: client not found")
	ErrServerNotStarted    = errors.New("photon: server not started")
	ErrMessageTooLarge     = errors.New("photon: message is too large")
	ErrUnsupportedObject   = errors.New("photon: unsupported object")
	ErrMalformedPacket     = errors.New("photon: malformed packet")
	ErrMalformedMessage    = errors.New("photon: malformed message")
	ErrUnexpectedPeer      = errors.New("photon: unexpected peer")
	ErrUnexpectedChallenge = errors.New("photon: unexpected challenge")
)

type ProtocolError struct {
	Kind   error
	Detail string
}

func (e *ProtocolError) Error() string {
	if e.Detail == "" {
		return e.Kind.Error()
	}
	return fmt.Sprintf("%v: %s", e.Kind, e.Detail)
}

func (e *ProtocolError) Unwrap() error { return e.Kind }
