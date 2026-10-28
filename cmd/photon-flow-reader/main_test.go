package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"example.com/brave-revival/src/photon"
	"github.com/stretchr/testify/require"
)

func flowPacket(kind byte, rsn uint32, payload []byte) []byte {
	data := make([]byte, 24+len(payload))
	data[3] = 1
	binary.BigEndian.PutUint32(data[8:12], 1)
	data[12] = kind
	if kind == 6 || kind == 8 {
		data[14] = 1
	}
	binary.BigEndian.PutUint32(data[16:20], uint32(12+len(payload)))
	binary.BigEndian.PutUint32(data[20:24], rsn)
	copy(data[24:], payload)
	return data
}

func TestWriteMessages(t *testing.T) {
	body, err := photon.MarshalOperationRequest(photon.OperationRequest{Code: 4, Parameters: photon.Parameters{}})
	require.NoError(t, err)
	content := append([]byte{243, byte(photon.MessageOperation)}, body...)
	fragment := func(number, offset, end uint32) []byte {
		payload := make([]byte, 20)
		for i, value := range []uint32{1, 2, number, uint32(len(content)), offset} {
			binary.BigEndian.PutUint32(payload[4*i:4*i+4], value)
		}
		return flowPacket(8, number+1, append(payload, content[offset:end]...))
	}
	first := fragment(1, 2, uint32(len(content)))
	last := fragment(0, 0, 2)
	response, err := photon.MarshalOperationResponse(photon.OperationResponse{Code: 5, Parameters: photon.Parameters{}})
	require.NoError(t, err)
	event, err := photon.MarshalEvent(photon.Event{Code: 6, Parameters: photon.Parameters{}})
	require.NoError(t, err)
	messages := []message{
		{fromClient: true, content: first, timestamp: 1},
		{content: flowPacket(6, 1, append([]byte{243, byte(photon.MessageOperationResponse)}, response...)), timestamp: 1.25},
		{fromClient: true, content: first, timestamp: 2},
		{fromClient: true, content: last, timestamp: 2.5},
		{fromClient: true, content: last, timestamp: 3},
		{content: flowPacket(7, 0, append([]byte{0, 0, 0, 1, 243, byte(photon.MessageEvent)}, event...)), timestamp: 3.25},
	}
	var buf bytes.Buffer
	require.NoError(t, writeMessages(&buf, messages))
	require.Equal(t, `<?xml version="1.0" encoding="UTF-8"?>
<Flow>
<Message From="Server" Timestamp="1970-01-01T00:00:01.25Z">
  <OperationResponse Code="5" ReturnCode="0"></OperationResponse>
</Message>
<Message From="Client" Timestamp="1970-01-01T00:00:02.5Z">
  <OperationRequest Code="4"></OperationRequest>
</Message>
<Message From="Server" Timestamp="1970-01-01T00:00:03.25Z">
  <Event Code="6"></Event>
</Message>
</Flow>
`, buf.String())
}

func TestWriteMessagesErrors(t *testing.T) {
	var buf bytes.Buffer
	err := writeMessages(&buf, []message{{fromClient: true, content: []byte{1}}})
	require.ErrorIs(t, err, photon.ErrMalformedPacket)
	require.Contains(t, err.Error(), "packet 1 from Client")

	writerError := errors.New("write failed")
	body, err := photon.MarshalEvent(photon.Event{Code: 6})
	require.NoError(t, err)
	err = writeMessages(failingWriter{writerError}, []message{{content: flowPacket(6, 1, append([]byte{243, byte(photon.MessageEvent)}, body...))}})
	require.ErrorIs(t, err, writerError)
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }
