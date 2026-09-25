package config

import (
	"encoding/json/v2"
	"fmt"
	"net/netip"
	"os"
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

	// AssetsDir is the local directory path where the Unity3D asset files are stored.
	//
	// The assets must be stored in the form `./ha/hash.unity3d`, where `ha` is the first two characters of the hash.
	AssetsDir string `json:"assets_dir"`

	// DBDir is the local directory path where the decoded game data are stored.
	DBDir string `json:"db_dir"`
}

func ReadFile(path string) (*Config, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(bytes, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.AdvertiseHost == "" {
		cfg.AdvertiseHost = cfg.Host.String()
	}
	if cfg.ChatPort == 0 {
		cfg.ChatPort = 10000
	}
	if cfg.PartyPort == 0 {
		cfg.PartyPort = 10002
	}
	if cfg.NotifyPort == 0 {
		cfg.NotifyPort = 10004
	}
	if cfg.PhotonPort == 0 {
		cfg.PhotonPort = 5056
	}

	return &cfg, nil
}
