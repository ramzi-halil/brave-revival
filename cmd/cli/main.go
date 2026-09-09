package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"example.com/brave-revival/src/config"
	"example.com/brave-revival/src/proxy"
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
	return eg.Wait()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: brave-revival <config.json>")
		os.Exit(1)
	}
	if err := run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
