# age-plugin-nostr

`age-plugin-nostr` is an age plugin that wraps age file keys to Nostr secp256k1
public keys. It accepts NIP-19 `npub` and `nsec` keys and converts them to the
age plugin encodings.

The plugin uses ephemeral secp256k1 ECDH and AES-256-GCM to wrap each age file
key. Its `nostr-v1` stanza is specific to this project and is not compatible
with NIP-44 or other Nostr encryption formats. Anyone with the Nostr secret
key can decrypt files addressed to its public key, so consider using a
dedicated Nostr key for age.

## Build

```sh
go install .
```

The installed executable must be named `age-plugin-nostr` and available in
`PATH` when using age.

## Generate a key

```sh
age-plugin-nostr keygen
```

The command prints a Nostr `npub` and `nsec`, plus the corresponding age
recipient and identity. Keep the `nsec` and age identity secret.

Convert existing NIP-19 keys:

```sh
age-plugin-nostr recipient npub1...
age-plugin-nostr identity nsec1...
```

## Encrypt and decrypt

```sh
age -r age1nostr1... -o secret.txt.age secret.txt
age-plugin-nostr identity nsec1... > nostr-identity.txt
age -d -i nostr-identity.txt secret.txt.age
```

The argument to `-i` is a file path, not the identity string. The identity
file must contain only the `AGE-PLUGIN-NOSTR-1...` value, with no `age identity:`
label or other keygen output. Keep this file secret.

The public key is an x-only 32-byte secp256k1 key. Identity and recipient
encodings use age's plugin Bech32 formats, while the conversion commands
handle NIP-19's `npub` and `nsec` formats.

## Integration test with rage

With Go and `rage` installed, run the encryption/decryption round-trip test:

```sh
./scripts/test-rage.sh
```

Set `RAGE` to use a `rage` executable at a nonstandard path.
