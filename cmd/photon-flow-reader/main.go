package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"time"

	"example.com/brave-revival/src/photon"

	"github.com/xleo1991/tnetstrings"
)

type message struct {
	fromClient bool
	content    []byte
	timestamp  float64
}

func parseFile(path string) ([]message, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var flow map[string]any
	if err = tnetstrings.NewDecoder(file).Decode(&flow); err != nil {
		return nil, err
	}

	rawMessages, ok := flow["messages"].([]any)
	if !ok {
		return nil, fmt.Errorf("invalid flow: messages is not a list")
	}

	messages := make([]message, 0, len(rawMessages))
	for _, msg := range rawMessages {
		entry, ok := msg.([]any)
		if !ok {
			return nil, fmt.Errorf("invalid message")
		}
		messages = append(messages, message{
			fromClient: entry[0].(bool),
			content:    []byte(entry[1].(string)),
			timestamp:  entry[2].(float64),
		})
	}

	return messages, nil
}

func writeMessages(w io.Writer, messages []message) error {
	if _, err := w.Write([]byte(xml.Header + "<Flow>\n")); err != nil {
		return err
	}

	var client, server photon.PacketDecoder
	e := xml.NewEncoder(w)
	e.Indent("", "  ")
	for i, msg := range messages {
		decoder, from := &server, "Server"
		if msg.fromClient {
			decoder, from = &client, "Client"
		}
		values, err := decoder.Decode(msg.content)
		if err != nil {
			return fmt.Errorf("packet %d from %s: %w", i+1, from, err)
		}
		seconds, fraction := math.Modf(msg.timestamp)
		timestamp := time.Unix(int64(seconds), int64(math.Round(fraction*1e9))).UTC().Format(time.RFC3339Nano)
		for _, value := range values {
			start := xml.StartElement{
				Name: xml.Name{Local: "Message"},
				Attr: []xml.Attr{
					{Name: xml.Name{Local: "From"}, Value: from},
					{Name: xml.Name{Local: "Timestamp"}, Value: timestamp},
				},
			}
			if err := e.EncodeToken(start); err != nil {
				return err
			}
			if err := e.Encode(value); err != nil {
				return err
			}
			if err := e.EncodeToken(start.End()); err != nil {
				return err
			}
		}
	}

	if err := e.Flush(); err != nil {
		return err
	}

	if _, err := w.Write([]byte("\n</Flow>\n")); err != nil {
		return err
	}
	return nil
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelInfo)

	if len(os.Args) < 2 {
		slog.Info("Usage: ./photon-flow-reader <udp.flow>")
		os.Exit(1)
	}

	messages, err := parseFile(os.Args[1])
	if err != nil {
		slog.Error("Error", "err", err)
		os.Exit(1)
	}

	if err := writeMessages(os.Stdout, messages); err != nil {
		slog.Error("Error", "err", err)
		os.Exit(1)
	}
}
