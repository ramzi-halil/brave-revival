package photon

import (
	"errors"
	"fmt"
)

const (
	TypeNull              byte = '*'
	TypeCustom            byte = 'c'
	TypeBoolean           byte = 'o'
	TypeByte              byte = 'b'
	TypeShort             byte = 'k'
	TypeInteger           byte = 'i'
	TypeLong              byte = 'l'
	TypeFloat             byte = 'f'
	TypeDouble            byte = 'd'
	TypeString            byte = 's'
	TypeEventData         byte = 'e'
	TypeHashtable         byte = 'h'
	TypeOperationResponse byte = 'p'
	TypeOperationRequest  byte = 'q'
	TypeArray             byte = 'y'
	TypeStringArray       byte = 'a'
	TypeByteArray         byte = 'x'
	TypeIntegerArray      byte = 'n'
	TypeObjectArray       byte = 'z'
	TypeDictionary        byte = 'D'
)

const (
	MessageInit                      byte = 0
	MessageInitResponse              byte = 1
	MessageOperation                 byte = 2
	MessageOperationResponse         byte = 3
	MessageEvent                     byte = 4
	MessageInternalOperation         byte = 6
	MessageInternalOperationResponse byte = 7
)

const (
	PeerIDServer      uint16 = 0
	PeerIDUnconnected uint16 = 0xffff
)

type ObjectDescriptor struct {
	Type       byte
	CustomType byte
	KeyType    *ObjectDescriptor
	ValueType  *ObjectDescriptor
}

type Object interface {
	photonType() ObjectDescriptor
	marshalPhoton(*objectWriter, int) error
}

type Parameters map[byte]Object

type Null struct{}
type Boolean bool
type Byte uint8
type Short uint16
type Integer uint32
type Long uint64
type Float float32
type Double float64
type String string
type ByteArray []byte
type IntegerArray []uint32
type StringArray []string
type ObjectArray []Object

type OperationRequest struct {
	Code       byte
	Parameters Parameters
	Internal   bool
}

type OperationResponse struct {
	Code         byte
	ReturnCode   int16
	DebugMessage Object
	Parameters   Parameters
	Internal     bool
}

type Event struct {
	Code       byte
	Parameters Parameters
}

type Custom struct {
	Code byte
	Data []byte
}

type Array struct {
	Type   ObjectDescriptor
	Values []Object
}

type Entry struct {
	Key   Object
	Value Object
}

type Hashtable []Entry

type Dictionary struct {
	KeyType   ObjectDescriptor
	ValueType ObjectDescriptor
	Entries   []Entry
}

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
