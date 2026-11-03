package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ohashi/age-plugin-nostr/internal/pngpayload"
)

func usage() {
	fmt.Fprintln(os.Stdout, `Usage:
  nostr-age-note -h | --help
  nostr-age-note embed <image.png> <file.age> <output.png>
  nostr-age-note extract <image.png> <output.age>

embed stores the binary age file in a private PNG chunk. The resulting PNG
can be uploaded and referenced from a Nostr note. extract restores the .age
file from the original PNG.`)
}

func run(args []string) error {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		usage()
		return nil
	}
	switch {
	case len(args) == 4 && args[0] == "embed":
		image, err := os.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("read PNG: %w", err)
		}
		payload, err := os.ReadFile(args[2])
		if err != nil {
			return fmt.Errorf("read age file: %w", err)
		}
		result, err := pngpayload.Embed(image, payload)
		if err != nil {
			return err
		}
		if err := os.WriteFile(args[3], result, 0600); err != nil {
			return fmt.Errorf("write PNG: %w", err)
		}
		return nil
	case len(args) == 3 && args[0] == "extract":
		image, err := os.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("read PNG: %w", err)
		}
		payload, err := pngpayload.Extract(image)
		if err != nil {
			return err
		}
		if err := os.WriteFile(args[2], payload, 0600); err != nil {
			return fmt.Errorf("write age file: %w", err)
		}
		return nil
	default:
		return errors.New("invalid arguments; run nostr-age-note --help")
	}
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
