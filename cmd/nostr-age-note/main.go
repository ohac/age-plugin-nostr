package main

import (
	"bytes"
	"compress/flate"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/ohashi/age-plugin-nostr/internal/pngpayload"
)

func usage() {
	fmt.Fprintln(os.Stdout, `Usage:
  nostr-age-note -h | --help
  nostr-age-note embed <file.age> [--format png|vs] [--cover image.png] [-o output]
  nostr-age-note extract <carrier> [-o output.age]

embed stores the binary age file in a PNG chunk (default) or invisible Unicode
variation selectors (vs). The default PNG cover is a small, simple image.
When -o is omitted, a timestamped filename is created in the current directory.

extract restores the .age file from either carrier. The extracted file also
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
			return errors.New("usage: nostr-age-note embed <file.age> [--format png|vs] [--cover image.png] [-o output]")
		}
		format := options["--format"]
		if format == "" {
			format = "png"
		}
		if format != "png" && format != "vs" {
			return fmt.Errorf("unsupported format %q (want png or vs)", format)
		}
		if format == "vs" && options["--cover"] != "" {
			return errors.New("--cover can only be used with --format png")
		}
		return embedFile(inputs[0], format, options["--cover"], outputPath(options, formatExtension(format)))
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
		if name == "-o" || name == "--output" || (allowCover && (name == "--cover" || name == "--format")) {
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

func formatExtension(format string) string {
	if format == "vs" {
		return "txt"
	}
	return "png"
}

func embedFile(agePath, format, coverPath, output string) error {
	if output == "" {
		return errors.New("could not choose an unused timestamped output filename")
	}
	payload, err := os.ReadFile(agePath)
	if err != nil {
		return fmt.Errorf("read age file: %w", err)
	}
	var result []byte
	if format == "vs" {
		result = encodeVariationSelectors(payload)
	} else {
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
		result, err = pngpayload.Embed(imageData, payload)
		if err != nil {
			return err
		}
	}
	if err := os.WriteFile(output, result, 0600); err != nil {
		return fmt.Errorf("write carrier: %w", err)
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
		return fmt.Errorf("read carrier: %w", err)
	}
	var payload []byte
	if pngpayload.IsPNG(imageData) {
		payload, err = pngpayload.Extract(imageData)
	} else {
		payload, err = decodeVariationSelectors(string(imageData))
	}
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
	palette := color.Palette{
		color.RGBA{R: 12, G: 19, B: 43, A: 255},
		color.RGBA{R: 71, G: 220, B: 205, A: 255},
		color.RGBA{R: 245, G: 247, B: 255, A: 255},
	}
	cover := image.NewPaletted(image.Rect(0, 0, 96, 96), palette)
	for y := 0; y < 96; y++ {
		for x := 0; x < 96; x++ {
			cover.SetColorIndex(x, y, 0)
		}
	}
	for y := 18; y < 79; y++ {
		for x := 29; x < 68; x++ {
			cover.SetColorIndex(x, y, 2)
		}
	}
	for y := 40; y < 72; y++ {
		for x := 36; x < 61; x++ {
			cover.SetColorIndex(x, y, 0)
		}
	}
	for y := 27; y < 43; y++ {
		for x := 39; x < 58; x++ {
			dx, dy := x-48, y-35
			if dx*dx+dy*dy <= 9*9 && dx*dx+dy*dy >= 6*6 {
				cover.SetColorIndex(x, y, 1)
			}
		}
	}
	for y := 40; y < 71; y++ {
		for x := 36; x < 61; x++ {
			cover.SetColorIndex(x, y, 1)
		}
	}
	for y := 47; y < 65; y++ {
		cover.SetColorIndex(48, y, 0)
	}
	var buffer bytes.Buffer
	encoder := png.Encoder{CompressionLevel: flate.BestCompression}
	if err := encoder.Encode(&buffer, cover); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func encodeVariationSelectors(payload []byte) []byte {
	var builder strings.Builder
	builder.Grow(4 + len(payload)*4)
	builder.WriteString("📦")
	for _, value := range payload {
		codepoint := rune(0xE0100 + int(value) - 16)
		if value < 16 {
			codepoint = rune(0xFE00 + int(value))
		}
		builder.WriteRune(codepoint)
	}
	return []byte(builder.String())
}

func decodeVariationSelectors(text string) ([]byte, error) {
	var payload []byte
	started := false
	for _, character := range text {
		codepoint := int(character)
		var value byte
		switch {
		case codepoint >= 0xFE00 && codepoint <= 0xFE0F:
			value = byte(codepoint - 0xFE00)
		case codepoint >= 0xE0100 && codepoint <= 0xE01EF:
			value = byte(codepoint - 0xE0100 + 16)
		default:
			if started {
				return payload, nil
			}
			continue
		}
		started = true
		payload = append(payload, value)
	}
	if len(payload) == 0 {
		return nil, errors.New("input contains no Unicode variation-selector payload")
	}
	return payload, nil
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
