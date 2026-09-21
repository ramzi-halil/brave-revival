package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/crow"
	"example.com/brave-revival/src/photon"
	"example.com/brave-revival/src/proxy"
	"example.com/brave-revival/src/chat"
	"golang.org/x/sync/errgroup"
)

func run() error {
	slog.Info("Welcome")

	cfgBytes, err := os.ReadFile(os.Args[1])
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	cfg, err := config.Parse(cfgBytes)
	if err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	eg, _ := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		return proxy.Run(cfg)
	})
	eg.Go(func() error {
		return photon.RunLobby(cfg)
	})
	eg.Go(func() error {
		return crow.RunParty(cfg)
	})
	eg.Go(func() error {
		return crow.RunNotify(cfg)
	})
	eg.Go(func() error {
		return chat.Run(cfg)
	})
	return eg.Wait()
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	if len(os.Args) < 2 {
		fmt.Println("Usage: brave-revival <config.json>")
		os.Exit(1)
	}
	if err := run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
