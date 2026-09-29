package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"example.com/brave-revival/src/chat"
	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/crow"
	"example.com/brave-revival/src/photon"
	"example.com/brave-revival/src/proxy"
	"golang.org/x/sync/errgroup"
)

func run() error {
	slog.Info("Welcome")

	cfg, err := config.ReadFile(os.Args[1])
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
	slog.SetLogLoggerLevel(slog.LevelInfo)

	if len(os.Args) < 2 {
		fmt.Println("Usage: ./brave-revival <config.json>")
		os.Exit(1)
	}
	if err := run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
