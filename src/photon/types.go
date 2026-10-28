package photon

import (
	"errors"
	"fmt"
)

type TypeCode byte

const (
	TypeNull              TypeCode = '*'
	TypeCustom            TypeCode = 'c'
	TypeBoolean           TypeCode = 'o'
	TypeByte              TypeCode = 'b'
	TypeShort             TypeCode = 'k'
	TypeInteger           TypeCode = 'i'
	TypeLong              TypeCode = 'l'
	TypeFloat             TypeCode = 'f'
	TypeDouble            TypeCode = 'd'
	TypeString            TypeCode = 's'
	TypeEventData         TypeCode = 'e'
	TypeHashtable         TypeCode = 'h'
	TypeOperationResponse TypeCode = 'p'
	TypeOperationRequest  TypeCode = 'q'
	TypeArray             TypeCode = 'y'
	TypeStringArray       TypeCode = 'a'
	TypeByteArray         TypeCode = 'x'
	TypeIntegerArray      TypeCode = 'n'
	TypeObjectArray       TypeCode = 'z'
	TypeDictionary        TypeCode = 'D'
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

type ObjectDescriptor struct {
	Type       TypeCode
	CustomType CustomCode
	KeyType    *ObjectDescriptor
	ValueType  *ObjectDescriptor
}

type Object interface {
	photonType() ObjectDescriptor
	marshalPhoton(*objectWriter, int) error
}

type Parameters map[ParamCode]Object

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
type IntegerArray []Integer
type StringArray []String
type ObjectArray []Object

type OperationRequest struct {
	Code       OpCode `xml:"Code,attr"`
	Parameters Parameters
	Internal   bool `xml:"Internal,attr,omitempty"`
}

type OperationResponse struct {
	Code         OpCode
	ReturnCode   int16
	DebugMessage Object
	Parameters   Parameters
	Internal     bool
}

type Event struct {
	Code       EvCode `xml:"Code,attr"`
	Parameters Parameters
}

type Custom struct {
	Code CustomCode
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
	KeyType   ObjectDescriptor `xml:"KeyType,attr"`
	ValueType ObjectDescriptor `xml:"ValueType,attr"`
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
