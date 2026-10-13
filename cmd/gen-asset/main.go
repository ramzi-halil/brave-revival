package main

import (
	"errors"
	"flag"
	"fmt"
	"image/png"
	"io"
	"os"
	"strings"
)

func run(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("gen-asset", flag.ContinueOnError)
	flags.SetOutput(stderr)
	input := flags.String("i", "", "input PNG file")
	output := flags.String("o", "", "output Unity3D asset bundle")
	platform := flags.String("p", "", "target platform (android or ios)")
	resource := flags.String("r", "", "resource name (e.g. 700101012)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *input == "" || *output == "" || *resource == "" || *platform == "" || flags.NArg() != 0 {
		return fmt.Errorf("usage: gen-asset -i input.png -o output.unity3d -p android|ios -r resource_name")
	}
	target := int32(0)
	switch *platform {
	case "ios":
		target = 9
	case "android":
		target = 13
	default:
		return fmt.Errorf("unsupported platform %q: expected android or ios", *platform)
	}
	if strings.TrimSpace(*resource) != *resource || strings.ContainsAny(*resource, "/\\\x00") {
		return fmt.Errorf("resource name must be a nonempty name without path separators or NUL")
	}
	file, err := os.Open(*input)
	if err != nil {
		return fmt.Errorf("open input: %w", err)
	}
	defer file.Close()
	config, err := png.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("read PNG: %w", err)
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > 16384 || config.Height > 16384 || uint64(config.Width)*uint64(config.Height) > (512<<20)/4 {
		return fmt.Errorf("PNG dimensions exceed the texture size limit")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind input: %w", err)
	}
	img, err := png.Decode(file)
	if err != nil {
		return fmt.Errorf("decode PNG: %w", err)
	}
	data, err := textureBundle(img, *resource, target)
	if err != nil {
		return err
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
