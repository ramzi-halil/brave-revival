package photon

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPacketDecoderMessages(t *testing.T) {
	for _, tt := range []struct {
		messageType byte
		value       Object
	}{
		{MessageOperation, OperationRequest{Code: 4, Parameters: Parameters{}}},
		{MessageInternalOperation, OperationRequest{Code: 0, Internal: true, Parameters: Parameters{}}},
		{MessageOperationResponse, OperationResponse{Code: 5, DebugMessage: Null{}, Parameters: Parameters{}}},
		{MessageInternalOperationResponse, OperationResponse{Code: 0, Internal: true, DebugMessage: Null{}, Parameters: Parameters{}}},
		{MessageEvent, Event{Code: 6, Parameters: Parameters{}}},
	} {
		body, err := MarshalObject(tt.value)
		require.NoError(t, err)
		content := append([]byte{243, tt.messageType}, body[1:]...)
		for _, encrypted := range []bool{false, true} {
			payload := content
			if encrypted {
				key := sha256.Sum256([]byte{1})
				block, err := aes.NewCipher(key[:])
				require.NoError(t, err)
				padding := aes.BlockSize - len(content[2:])%aes.BlockSize
				plain := append(bytes.Clone(content[2:]), bytes.Repeat([]byte{byte(padding)}, padding)...)
				cipher.NewCBCEncrypter(block, make([]byte, aes.BlockSize)).CryptBlocks(plain, plain)
				payload = append([]byte{243, tt.messageType | 0x80}, plain...)
			}
			data, err := marshalPacket(packet{commands: []command{
				{typeCode: commandACK},
				{typeCode: commandReliable, rsn: 9, flags: commandFlagReliable, payload: []byte{243, MessageInit}},
				{typeCode: commandReliable, rsn: 10, flags: commandFlagReliable, payload: payload},
				{typeCode: commandReliable, rsn: 11, flags: commandFlagReliable, payload: payload},
			}})
			require.NoError(t, err)
			var decoder PacketDecoder
			values, err := decoder.Decode(data)
			require.NoError(t, err)
			require.Equal(t, []Object{tt.value, tt.value}, values)
			values, err = decoder.Decode(data)
			require.NoError(t, err)
			require.Empty(t, values)
		}
	}
}

func TestPacketDecoderFragmentChannels(t *testing.T) {
	body, err := MarshalEvent(Event{Code: 7})
	require.NoError(t, err)
	content := append([]byte{243, MessageEvent}, body...)
	var decoder PacketDecoder
	for i, step := range []struct {
		channel byte
		number  uint32
	}{
		{0, 1}, {1, 0}, {1, 1}, {0, 0},
	} {
		offset, end := uint32(0), uint32(2)
		if step.number == 1 {
			offset, end = 2, uint32(len(content))
		}
		payload := make([]byte, 20)
		for i, value := range []uint32{1, 2, step.number, uint32(len(content)), offset} {
			binary.BigEndian.PutUint32(payload[4*i:4*i+4], value)
		}
		payload = append(payload, content[offset:end]...)
		data, err := marshalPacket(packet{commands: []command{{typeCode: commandFragment, channel: step.channel, rsn: step.number + 1, flags: commandFlagReliable, payload: payload}}})
		require.NoError(t, err)
		values, err := decoder.Decode(data)
		require.NoError(t, err)
		if i >= 2 {
			require.Equal(t, []Object{Event{Code: 7, Parameters: Parameters{}}}, values)
		} else {
			require.Empty(t, values)
		}
	}
}

func TestPacketDecoderErrors(t *testing.T) {
	for _, cmd := range []command{
		{typeCode: commandReliable},
		{typeCode: commandReliable, payload: []byte{243, MessageOperation}},
		{typeCode: commandReliable, payload: []byte{243, 99}},
		{typeCode: commandUnreliable, payload: []byte{1}},
		{typeCode: commandFragment, payload: make([]byte, 20)},
		{typeCode: 99},
	} {
		data, err := marshalPacket(packet{commands: []command{cmd}})
		require.NoError(t, err)
		var decoder PacketDecoder
		_, err = decoder.Decode(data)
		require.Error(t, err)
	}
}
