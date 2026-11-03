package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"filippo.io/age/plugin"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/bech32"
)

const pluginName = "nostr"
const stanzaType = "nostr-v1"

type nostrRecipient struct{ pub *btcec.PublicKey }
type nostrIdentity struct{ priv *btcec.PrivateKey }

func (r *nostrRecipient) Wrap(fileKey []byte) ([]*age.Stanza, error) {
	ephemeral, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}
	shared := btcec.GenerateSharedSecret(ephemeral, r.pub)
	key := deriveKey(shared)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	sealed := aead.Seal(nil, nonce, fileKey, []byte(stanzaType))
	body := append(nonce, sealed...)
	return []*age.Stanza{{Type: stanzaType, Args: []string{base64.RawStdEncoding.EncodeToString(ephemeral.PubKey().SerializeCompressed())}, Body: body}}, nil
}

func (i *nostrIdentity) Unwrap(stanzas []*age.Stanza) ([]byte, error) {
	for _, stanza := range stanzas {
		if stanza.Type != stanzaType || len(stanza.Args) != 1 {
			continue
		}
		ephBytes, err := base64.RawStdEncoding.DecodeString(stanza.Args[0])
		if err != nil {
			continue
		}
		eph, err := btcec.ParsePubKey(ephBytes)
		if err != nil {
			continue
		}
		shared := btcec.GenerateSharedSecret(i.priv, eph)
		key := deriveKey(shared)
		block, err := aes.NewCipher(key)
		if err != nil {
			continue
		}
		aead, err := cipher.NewGCM(block)
		if err != nil || len(stanza.Body) < aead.NonceSize() {
			continue
		}
		plain, err := aead.Open(nil, stanza.Body[:aead.NonceSize()], stanza.Body[aead.NonceSize():], []byte(stanzaType))
		if err == nil {
			return plain, nil
		}
	}
	return nil, age.ErrIncorrectIdentity
}

func deriveKey(shared []byte) []byte {
	h := sha256.New()
	h.Write([]byte("age-plugin-nostr-v1\x00"))
	h.Write(shared)
	return h.Sum(nil)
}

func parseRecipient(data []byte) (age.Recipient, error) {
	if len(data) != 32 {
		return nil, errors.New("Nostr public key must be 32 bytes")
	}
	pub, err := btcec.ParsePubKey(append([]byte{2}, data...))
	if err != nil {
		return nil, fmt.Errorf("invalid x-only public key: %w", err)
	}
	return &nostrRecipient{pub: pub}, nil
}

func parseIdentity(data []byte) (age.Identity, error) {
	if len(data) != 32 {
		return nil, errors.New("Nostr secret key must be 32 bytes")
	}
	var scalar btcec.ModNScalar
	if scalar.SetByteSlice(data) || scalar.IsZero() {
		return nil, errors.New("invalid Nostr secret key")
	}
	priv, pub := btcec.PrivKeyFromBytes(data)
	if pub.SerializeCompressed()[0] == 3 {
		priv.Key.Negate()
	}
	return &nostrIdentity{priv: priv}, nil
}

func decodeNIP19(value, prefix string) ([]byte, error) {
	hrp, data, err := bech32.Decode(value)
	if err != nil {
		return nil, err
	}
	if hrp != prefix {
		return nil, fmt.Errorf("expected %s key", prefix)
	}
	decoded, err := bech32.ConvertBits(data, 5, 8, false)
	if err != nil {
		return nil, err
	}
	if len(decoded) != 32 {
		return nil, errors.New("key must decode to 32 bytes")
	}
	return decoded, nil
}

func decodeSecretKey(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	var secret []byte
	var err error
	if strings.HasPrefix(strings.ToLower(value), "nsec1") {
		secret, err = decodeNIP19(strings.ToLower(value), "nsec")
	} else {
		value = strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
		secret, err = hex.DecodeString(value)
		if err == nil && len(secret) != 32 {
			err = errors.New("hex secret key must contain exactly 64 hexadecimal characters")
		}
	}
	if err != nil {
		return nil, err
	}
	var scalar btcec.ModNScalar
	if scalar.SetByteSlice(secret) || scalar.IsZero() {
		return nil, errors.New("invalid Nostr secret key")
	}
	return secret, nil
}

func encodeNIP19(prefix string, key []byte) (string, error) {
	data, err := bech32.ConvertBits(key, 8, 5, true)
	if err != nil {
		return "", err
	}
	return bech32.Encode(prefix, data)
}

func keygen() error {
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		return err
	}
	secret := priv.Serialize()
	pub := priv.PubKey().SerializeCompressed()[1:]
	nsec, err := encodeNIP19("nsec", secret)
	if err != nil {
		return err
	}
	npub, err := encodeNIP19("npub", pub)
	if err != nil {
		return err
	}
	fmt.Printf("Nostr public key: %s\n", npub)
	fmt.Printf("age recipient: %s\n", plugin.EncodeRecipient(pluginName, pub))
	fmt.Printf("Nostr secret key: %s\n", nsec)
	fmt.Printf("age identity: %s\n", plugin.EncodeIdentity(pluginName, secret))
	return nil
}

func convert(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: age-plugin-nostr recipient <npub> | identity <nsec-or-hex> | identity-file <path>")
	}
	switch args[0] {
	case "recipient":
		key, err := decodeNIP19(strings.ToLower(args[1]), "npub")
		if err != nil {
			return err
		}
		fmt.Println(plugin.EncodeRecipient(pluginName, key))
	case "identity":
		key, err := decodeSecretKey(args[1])
		if err != nil {
			return err
		}
		fmt.Println(plugin.EncodeIdentity(pluginName, key))
	case "identity-file":
		contents, err := os.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("read secret key file: %w", err)
		}
		key, err := decodeSecretKey(string(contents))
		if err != nil {
			return err
		}
		fmt.Println(plugin.EncodeIdentity(pluginName, key))
	default:
		return errors.New("usage: age-plugin-nostr recipient <npub> | identity <nsec-or-hex> | identity-file <path>")
	}
	return nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `Usage:
  age-plugin-nostr -h | --help
  age-plugin-nostr keygen
  age-plugin-nostr recipient <npub>
  age-plugin-nostr identity <nsec-or-hex>
  age-plugin-nostr identity-file <path>

Commands:
  keygen                 Generate a Nostr key and age recipient/identity
  recipient <npub>       Convert a NIP-19 public key to an age recipient
  identity <key>         Convert an nsec or 32-byte hex secret to an age identity
  identity-file <path>   Read an nsec or hex secret file and print an age identity

The identity and identity-file commands print secret age identity material.
Keep that output private. For age decryption, write the output to a file and
pass the file path with -i/--identity.`)
}

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "-h" || arg == "--help" {
			printUsage(os.Stdout)
			return
		}
	}
	if len(os.Args) > 1 && (os.Args[1] == "keygen" || os.Args[1] == "recipient" || os.Args[1] == "identity" || os.Args[1] == "identity-file") {
		var err error
		switch os.Args[1] {
		case "keygen":
			err = keygen()
		case "recipient", "identity", "identity-file":
			err = convert(os.Args[1:])
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	p, err := plugin.New(pluginName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	p.HandleRecipient(parseRecipient)
	p.HandleIdentity(parseIdentity)
	os.Exit(p.Main())
}
