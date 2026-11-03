package main

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"filippo.io/age"
	"filippo.io/age/plugin"
	"github.com/btcsuite/btcd/btcec/v2"
)

func TestWrapUnwrapWithNostrKey(t *testing.T) {
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	secret := priv.Serialize()
	public := priv.PubKey().SerializeCompressed()[1:]

	recipient, err := parseRecipient(public)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := parseIdentity(secret)
	if err != nil {
		t.Fatal(err)
	}
	fileKey := []byte("0123456789abcdef")
	stanzas, err := recipient.(*nostrRecipient).Wrap(fileKey)
	if err != nil {
		t.Fatal(err)
	}
	got, err := identity.(*nostrIdentity).Unwrap(stanzas)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, fileKey) {
		t.Fatalf("unwrapped key %x, want %x", got, fileKey)
	}
}

func TestWrapUnwrapNormalizesOddYSecret(t *testing.T) {
	var priv *btcec.PrivateKey
	for {
		var err error
		priv, err = btcec.NewPrivateKey()
		if err != nil {
			t.Fatal(err)
		}
		if priv.PubKey().SerializeCompressed()[0] == 3 {
			break
		}
	}
	public := priv.PubKey().SerializeCompressed()[1:]
	recipient, err := parseRecipient(public)
	if err != nil {
		t.Fatal(err)
	}
	identity, err := parseIdentity(priv.Serialize())
	if err != nil {
		t.Fatal(err)
	}
	fileKey := []byte("0123456789abcdef")
	stanzas, err := recipient.(*nostrRecipient).Wrap(fileKey)
	if err != nil {
		t.Fatal(err)
	}
	got, err := identity.(*nostrIdentity).Unwrap(stanzas)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, fileKey) {
		t.Fatalf("unwrapped key %x, want %x", got, fileKey)
	}
}

func TestNIP19KeyConversion(t *testing.T) {
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	secret := priv.Serialize()
	public := priv.PubKey().SerializeCompressed()[1:]

	nsec, err := encodeNIP19("nsec", secret)
	if err != nil {
		t.Fatal(err)
	}
	npub, err := encodeNIP19("npub", public)
	if err != nil {
		t.Fatal(err)
	}
	gotSecret, err := decodeNIP19(nsec, "nsec")
	if err != nil {
		t.Fatal(err)
	}
	gotPublic, err := decodeNIP19(npub, "npub")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotSecret, secret) || !bytes.Equal(gotPublic, public) {
		t.Fatal("NIP-19 round trip changed key bytes")
	}
	if _, err := decodeNIP19(npub, "nsec"); err == nil {
		t.Fatal("accepted npub where nsec was required")
	}
	if got := plugin.EncodeRecipient(pluginName, gotPublic); got == "" {
		t.Fatal("failed to encode age recipient")
	}
}

func TestDecodeSecretKeyFormats(t *testing.T) {
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	secret := priv.Serialize()
	nsec, err := encodeNIP19("nsec", secret)
	if err != nil {
		t.Fatal(err)
	}
	hexKey := hex.EncodeToString(secret)

	for _, input := range []string{nsec, hexKey, "0x" + hexKey, "  " + hexKey + "\n"} {
		got, err := decodeSecretKey(input)
		if err != nil {
			t.Fatalf("decodeSecretKey failed: %v", err)
		}
		if !bytes.Equal(got, secret) {
			t.Fatalf("decoded key %x, want %x", got, secret)
		}
	}

	for _, input := range []string{"", "0x1234", hex.EncodeToString(make([]byte, 32))} {
		if _, err := decodeSecretKey(input); err == nil {
			t.Fatal("decodeSecretKey unexpectedly accepted an invalid key")
		}
	}
}

func TestPrintUsage(t *testing.T) {
	var output bytes.Buffer
	printUsage(&output)
	for _, want := range []string{"Usage:", "-h | --help", "keygen", "recipient <npub>", "identity <nsec-or-hex>", "identity-file <path>"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("usage output does not contain %q", want)
		}
	}
}

func TestUnwrapRejectsWrongIdentityAndTampering(t *testing.T) {
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	other, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := parseRecipient(priv.PubKey().SerializeCompressed()[1:])
	if err != nil {
		t.Fatal(err)
	}
	stanzas, err := recipient.(*nostrRecipient).Wrap([]byte("0123456789abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	wrongIdentity, err := parseIdentity(other.Serialize())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wrongIdentity.(*nostrIdentity).Unwrap(stanzas); err == nil {
		t.Fatal("wrong identity unexpectedly unwrapped stanza")
	}

	corrupted := &age.Stanza{Type: stanzas[0].Type, Args: append([]string(nil), stanzas[0].Args...), Body: append([]byte(nil), stanzas[0].Body...)}
	corrupted.Body[len(corrupted.Body)-1] ^= 1
	identity, err := parseIdentity(priv.Serialize())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := identity.(*nostrIdentity).Unwrap([]*age.Stanza{corrupted}); err == nil {
		t.Fatal("tampered stanza unexpectedly unwrapped")
	}
}
