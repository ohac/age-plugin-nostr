package main

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/ohashi/age-plugin-nostr/internal/pngpayload"
)

func usage() {
	fmt.Fprintln(os.Stdout, `Usage:
  nostr-age-note -h | --help
  nostr-age-note embed <file.age> [--cover image.png] [-o output.png]
  nostr-age-note extract <image.png> [-o output.age]

embed stores the binary age file in a private PNG chunk. Without --cover, it
generates a dark Nostr-themed cover showing an encrypted file and lock. When
-o is omitted, a timestamped filename is created in the current directory.
Upload the resulting PNG and reference its URL from a Nostr note.

extract restores the .age file from the original PNG. The extracted file also
gets a timestamped filename when -o is omitted.`)
}

func run(args []string) error {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		usage()
		return nil
	}
	if len(args) == 0 {
		return errors.New("missing command; run nostr-age-note --help")
	}
	switch args[0] {
	case "embed":
		inputs, options, help, err := parseOptions(args[1:], true)
		if err != nil || help {
			return err
		}
		if len(inputs) != 1 {
			return errors.New("usage: nostr-age-note embed <file.age> [--cover image.png] [-o output.png]")
		}
		return embedFile(inputs[0], options["--cover"], outputPath(options, "png"))
	case "extract":
		inputs, options, help, err := parseOptions(args[1:], false)
		if err != nil || help {
			return err
		}
		if len(inputs) != 1 {
			return errors.New("usage: nostr-age-note extract <image.png> [-o output.age]")
		}
		return extractFile(inputs[0], outputPath(options, "age"))
	default:
		return errors.New("invalid command; run nostr-age-note --help")
	}
}

func parseOptions(args []string, allowCover bool) ([]string, map[string]string, bool, error) {
	inputs := make([]string, 0, 1)
	options := make(map[string]string)
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if arg == "-h" || arg == "--help" {
			usage()
			return nil, nil, true, nil
		}
		name, value, hasValue := strings.Cut(arg, "=")
		if name == "-o" || name == "--output" || (allowCover && name == "--cover") {
			if !hasValue {
				index++
				if index == len(args) {
					return nil, nil, false, fmt.Errorf("option %s requires a value", name)
				}
				value = args[index]
			}
			if value == "" {
				return nil, nil, false, fmt.Errorf("option %s requires a non-empty value", name)
			}
			key := name
			if name == "-o" {
				key = "--output"
			}
			if options[key] != "" {
				return nil, nil, false, fmt.Errorf("option %s was specified more than once", key)
			}
			options[key] = value
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return nil, nil, false, fmt.Errorf("unknown option %s", arg)
		}
		inputs = append(inputs, arg)
	}
	return inputs, options, false, nil
}

func outputPath(options map[string]string, extension string) string {
	if output := options["--output"]; output != "" {
		return output
	}
	path, err := defaultOutputName(extension)
	if err != nil {
		return ""
	}
	return path
}

func embedFile(agePath, coverPath, output string) error {
	if output == "" {
		return errors.New("could not choose an unused timestamped output filename")
	}
	payload, err := os.ReadFile(agePath)
	if err != nil {
		return fmt.Errorf("read age file: %w", err)
	}
	var imageData []byte
	if coverPath != "" {
		imageData, err = os.ReadFile(coverPath)
		if err != nil {
			return fmt.Errorf("read cover PNG: %w", err)
		}
	} else {
		imageData, err = generatedCover()
		if err != nil {
			return fmt.Errorf("generate cover PNG: %w", err)
		}
	}
	result, err := pngpayload.Embed(imageData, payload)
	if err != nil {
		return err
	}
	if err := os.WriteFile(output, result, 0600); err != nil {
		return fmt.Errorf("write PNG: %w", err)
	}
	fmt.Println(output)
	return nil
}

func extractFile(imagePath, output string) error {
	if output == "" {
		return errors.New("could not choose an unused timestamped output filename")
	}
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return fmt.Errorf("read PNG: %w", err)
	}
	payload, err := pngpayload.Extract(imageData)
	if err != nil {
		return err
	}
	if err := os.WriteFile(output, payload, 0600); err != nil {
		return fmt.Errorf("write age file: %w", err)
	}
	fmt.Println(output)
	return nil
}

func defaultOutputName(extension string) (string, error) {
	stamp := time.Now().Format("20060102-150405")
	for suffix := 0; suffix < 10000; suffix++ {
		name := fmt.Sprintf("nostr-age-%s.%s", stamp, extension)
		if suffix > 0 {
			name = fmt.Sprintf("nostr-age-%s-%02d.%s", stamp, suffix, extension)
		}
		_, err := os.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			return name, nil
		}
		if err != nil {
			return "", fmt.Errorf("check output path: %w", err)
		}
	}
	return "", errors.New("could not choose an unused timestamped output filename")
}

func generatedCover() ([]byte, error) {
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, makeCoverImage()); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
