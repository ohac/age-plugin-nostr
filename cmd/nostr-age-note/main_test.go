package main

import (
	"bytes"
	"image/png"
	"testing"
)

func TestVariationSelectorRoundTrip(t *testing.T) {
	payload := make([]byte, 256)
	for index := range payload {
		payload[index] = byte(index)
	}
	encoded := encodeVariationSelectors(payload)
	decoded, err := decodeVariationSelectors(string(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatal("variation-selector round trip changed payload")
	}
}

func TestVariationSelectorRejectsMissingPayload(t *testing.T) {
	if _, err := decodeVariationSelectors("📦 ordinary text"); err == nil {
		t.Fatal("accepted text without variation selectors")
	}
}

func TestGeneratedCoverIsSmallAndValidPNG(t *testing.T) {
	cover, err := generatedCover()
	if err != nil {
		t.Fatal(err)
	}
	if len(cover) > 1024 {
		t.Fatalf("generated cover is unexpectedly large: %d bytes", len(cover))
	}
	if _, err := png.Decode(bytes.NewReader(cover)); err != nil {
		t.Fatalf("generated cover is not a valid PNG: %v", err)
	}
}
