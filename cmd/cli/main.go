package main

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"os"

	"example.com/brave-revival/src/proxy"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	Proxy proxy.Config `json:"proxy"`
}

func run() error {
	f, err := os.Open(os.Args[1])
	if err != nil {
		return fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	var config Config
	if err := json.UnmarshalRead(f, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	eg, _ := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		return proxy.Run(&config.Proxy)
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
