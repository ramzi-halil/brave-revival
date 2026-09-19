package photon

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	commandACK           byte = 1
	commandConnect       byte = 2
	commandVerifyConnect byte = 3
	commandDisconnect    byte = 4
	commandPing          byte = 5
	commandReliable      byte = 6
	commandUnreliable    byte = 7
	commandFragment      byte = 8
	commandServerTime    byte = 12

	commandFlagReliable    byte = 1
	commandFlagUnsequenced byte = 2

	packetHeaderSize  = 12
	commandHeaderSize = 12
	defaultMTU        = 1200
	maxMessageSize    = 16 << 20
)

type packet struct {
	peerID    uint16
	flags     byte
	sentTime  uint32
	challenge uint32
	commands  []command
}

type command struct {
	typeCode byte
	channel  byte
	flags    byte
	reserved byte
	rsn      uint32
	payload  []byte
}

func parsePacket(data []byte) (packet, error) {
	if len(data) < packetHeaderSize {
		return packet{}, protocolError(ErrMalformedPacket, "packet header is truncated")
	}
	p := packet{
		peerID:    binary.BigEndian.Uint16(data[0:2]),
		flags:     data[2],
		sentTime:  binary.BigEndian.Uint32(data[4:8]),
		challenge: binary.BigEndian.Uint32(data[8:12]),
	}
	if p.flags != 0 {
		return packet{}, protocolError(ErrMalformedPacket, fmt.Sprintf("unsupported flags %#x", p.flags))
	}
	count := int(data[3])
	offset := packetHeaderSize
	p.commands = make([]command, 0, count)
	for range count {
		if len(data)-offset < commandHeaderSize {
			return packet{}, protocolError(ErrMalformedPacket, "command header is truncated")
		}
		length := int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
		if length < commandHeaderSize || length > len(data)-offset {
			return packet{}, protocolError(ErrMalformedPacket, "invalid command length")
		}
		cmd := command{
			typeCode: data[offset],
			channel:  data[offset+1],
			flags:    data[offset+2],
			reserved: data[offset+3],
			rsn:      binary.BigEndian.Uint32(data[offset+8 : offset+12]),
			payload:  append([]byte(nil), data[offset+12:offset+length]...),
		}
		if cmd.flags & ^byte(commandFlagReliable|commandFlagUnsequenced) != 0 {
			return packet{}, protocolError(ErrMalformedPacket, "unsupported command flags")
		}
		p.commands = append(p.commands, cmd)
		offset += length
	}
	if offset != len(data) {
		return packet{}, protocolError(ErrMalformedPacket, "trailing packet data")
	}
	return p, nil
}

func marshalPacket(p packet) ([]byte, error) {
	if len(p.commands) > 255 {
		return nil, ErrMessageTooLarge
	}
	var buf bytes.Buffer
	writeUint16(&buf, p.peerID)
	buf.WriteByte(p.flags)
	buf.WriteByte(byte(len(p.commands)))
	writeUint32(&buf, p.sentTime)
	writeUint32(&buf, p.challenge)
	for _, cmd := range p.commands {
		if len(cmd.payload) > int(^uint32(0))-commandHeaderSize {
			return nil, ErrMessageTooLarge
		}
		buf.WriteByte(cmd.typeCode)
		buf.WriteByte(cmd.channel)
		buf.WriteByte(cmd.flags)
		buf.WriteByte(cmd.reserved)
		writeUint32(&buf, uint32(commandHeaderSize+len(cmd.payload)))
		writeUint32(&buf, cmd.rsn)
		buf.Write(cmd.payload)
	}
	return buf.Bytes(), nil
}

func ackCommand(channel byte, rsn, sentTime uint32) command {
	payload := make([]byte, 8)
	binary.BigEndian.PutUint32(payload[0:4], rsn)
	binary.BigEndian.PutUint32(payload[4:8], sentTime)
	return command{typeCode: commandACK, channel: channel, payload: payload}
}
