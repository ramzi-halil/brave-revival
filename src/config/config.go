package config

import (
	"encoding/json/v2"
	"fmt"
	"net/netip"
)

type Config struct {
	// Host is the IP address of the server should bind to.
	//
	// Use `::` to bind to all interfaces.
	//
	// This program always use UDP port 5055 for the Photon lobby server. Other ports can be freely configured.
	Host netip.Addr `json:"host"`

	// AdvertiseHost is the domain name which the client is expected to connect to.
	AdvertiseHost string `json:"advertise_host"`

	// ProxyPort is the TCP (HTTP) port number of the proxy server which the client is expected to connect to.
	ProxyPort uint16 `json:"proxy_port"`

	// ChatPort is the TCP (WebSocket) port number of the chat server.
	//
	// The standard server uses port 10000 for this.
	ChatPort uint16 `json:"chat_port"`

	// PartyPort is the TCP (gRPC/H2C) port number of the party server.
	//
	// The standard server uses port 10002 for this.
	PartyPort uint16 `json:"party_port"`

	// NotifyPort is the TCP (gRPC/H2C) port number of the notify server.
	//
	// The standard server uses port 10004 for this.
	NotifyPort uint16 `json:"notify_port"`

	// PhotonPort is the UDP port number of the Photon game server.
	//
	// The standard server uses port 5056 for this.
	PhotonPort uint16 `json:"photon_port"`

	// TLSDir is the local directory path where the TLS certificate and key are stored.
	TLSDir string `json:"tls_dir"`

	// Assets is the local directory path where the Unity3D asset files are stored.
	//
	// The assets must be stored in the form `./ha/hash.unity3d`, where `ha` is the first two characters of the hash.
	Assets string `json:"assets"`
}

func Parse(bytes []byte) (*Config, error) {
	var cfg Config
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.AdvertiseHost == "" {
		cfg.AdvertiseHost = cfg.Host.String()
	}

	return &cfg, nil
}
