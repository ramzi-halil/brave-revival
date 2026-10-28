package photon

import (
	"encoding/binary"
	"fmt"
)

// PacketDecoder decodes one direction of a captured conversation, in capture order.
// Its zero value is ready to use. Use separate decoders for the two directions.
type PacketDecoder struct {
	challenge uint32
	seen      map[pendingKey]struct{}
	recvUSN   map[byte]uint32
	fragments map[byte]map[uint32]*fragmentSet
}

// Decode ignores transport and initialization messages and suppresses retransmissions.
// Fragmented messages are returned with the packet that completes their reassembly.
func (d *PacketDecoder) Decode(data []byte) ([]Object, error) {
	p, err := parsePacket(data)
	if err != nil {
		return nil, err
	}
	if d.seen == nil || d.challenge != p.challenge {
		*d = PacketDecoder{
			challenge: p.challenge,
			seen:      make(map[pendingKey]struct{}),
			recvUSN:   make(map[byte]uint32),
			fragments: make(map[byte]map[uint32]*fragmentSet),
		}
	}
	var messages []Object
	for _, cmd := range p.commands {
		key := pendingKey{channel: cmd.channel, rsn: cmd.rsn}
		if cmd.flags&commandFlagReliable != 0 {
			if _, seen := d.seen[key]; seen {
				continue
			}
		}
		var content []byte
		switch cmd.typeCode {
		case commandACK, commandConnect, commandVerifyConnect, commandDisconnect, commandPing, commandServerTime:
			continue
		case commandReliable:
			content = cmd.payload
		case commandUnreliable:
			if len(cmd.payload) < 4 {
				return nil, protocolError(ErrMalformedPacket, "unreliable payload is truncated")
			}
			usn := binary.BigEndian.Uint32(cmd.payload[:4])
			if last, exists := d.recvUSN[cmd.channel]; exists && usn <= last {
				continue
			}
			d.recvUSN[cmd.channel] = usn
			content = cmd.payload[4:]
		case commandFragment:
			sets := d.fragments[cmd.channel]
			if sets == nil {
				sets = make(map[uint32]*fragmentSet)
				d.fragments[cmd.channel] = sets
			}
			content, err = assembleFragment(sets, cmd.payload)
			if err != nil {
				return nil, err
			}
		default:
			return nil, protocolError(ErrMalformedPacket, fmt.Sprintf("unsupported command type %d", cmd.typeCode))
		}
		if content != nil || cmd.typeCode != commandFragment {
			value, err := decodeMessage(content)
			if err != nil {
				return nil, err
			}
			if value != nil {
				messages = append(messages, value)
			}
		}
		if cmd.flags&commandFlagReliable != 0 {
			d.seen[key] = struct{}{}
		}
	}
	return messages, nil
}

func decodeMessage(content []byte) (Object, error) {
	if len(content) < 2 || content[0] != 243 {
		return nil, protocolError(ErrMalformedMessage, "invalid content signature")
	}
	messageType := MsgCode(content[1])
	body := content[2:]
	if messageType&0x80 != 0 {
		messageType &^= 0x80
		var err error
		body, err = decryptMessage(body)
		if err != nil {
			return nil, err
		}
	}
	switch messageType {
	case MessageInit, MessageInitResponse:
		return nil, nil
	case MessageOperation, MessageInternalOperation:
		request, err := UnmarshalOperationRequest(body)
		request.Internal = messageType == MessageInternalOperation
		return request, err
	case MessageOperationResponse, MessageInternalOperationResponse:
		response, err := UnmarshalOperationResponse(body)
		response.Internal = messageType == MessageInternalOperationResponse
		return response, err
	case MessageEvent:
		return UnmarshalEvent(body)
	default:
		return nil, protocolError(ErrMalformedMessage, fmt.Sprintf("unsupported message type %d", messageType))
	}
}

func assembleFragment(sets map[uint32]*fragmentSet, payload []byte) ([]byte, error) {
	if len(payload) < 20 {
		return nil, protocolError(ErrMalformedPacket, "fragment payload is truncated")
	}
	start := binary.BigEndian.Uint32(payload[0:4])
	count := binary.BigEndian.Uint32(payload[4:8])
	number := binary.BigEndian.Uint32(payload[8:12])
	total := binary.BigEndian.Uint32(payload[12:16])
	offset := binary.BigEndian.Uint32(payload[16:20])
	content := payload[20:]
	if count == 0 || number >= count || total > maxMessageSize || count > total || len(content) == 0 || offset > total || uint64(offset)+uint64(len(content)) > uint64(total) {
		return nil, protocolError(ErrMalformedPacket, "invalid fragment bounds")
	}
	set := sets[start]
	if set == nil {
		set = &fragmentSet{count: count, total: total, parts: make(map[uint32]fragmentPart)}
		sets[start] = set
	}
	if set.count != count || set.total != total {
		return nil, protocolError(ErrMalformedPacket, "inconsistent fragment metadata")
	}
	if _, exists := set.parts[number]; !exists {
		set.parts[number] = fragmentPart{offset: offset, data: append([]byte(nil), content...)}
		set.received++
	}
	if set.received != set.count {
		return nil, nil
	}
	delete(sets, start)
	assembled := make([]byte, set.total)
	covered := make([]bool, set.total)
	for _, part := range set.parts {
		copy(assembled[part.offset:], part.data)
		for i := part.offset; i < part.offset+uint32(len(part.data)); i++ {
			if covered[i] {
				return nil, protocolError(ErrMalformedPacket, "overlapping fragments")
			}
			covered[i] = true
		}
	}
	for _, present := range covered {
		if !present {
			return nil, protocolError(ErrMalformedPacket, "fragment set has gaps")
		}
	}
	return assembled, nil
}
