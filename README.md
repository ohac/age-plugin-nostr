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
age-plugin-nostr identity 0123456789abcdef... # 64 hexadecimal characters
age-plugin-nostr identity-file ~/.nostr/nsec > nostr-identity.txt
```

`identity` accepts either an `nsec1...` value or a 32-byte secret key in
64-character hexadecimal form, optionally prefixed by `0x`. `identity-file`
reads either representation from a file and prints the age identity encoding.
This converts the secret key file; it does not modify the input file.

## Encrypt and decrypt

```sh
age -r age1nostr1... -o secret.txt.age secret.txt
age-plugin-nostr identity-file ~/.nostr/nsec > nostr-identity.txt
age -d -i nostr-identity.txt secret.txt.age
```

The argument to `-i` is a file path, not the identity string. `rage` and age
clients expect the file to contain the `AGE-PLUGIN-NOSTR-1...` encoding; they
do not automatically convert raw `nsec1...` or hex secret key files. Keep the
resulting identity file secret.

The public key is an x-only 32-byte secp256k1 key. Identity and recipient
encodings use age's plugin Bech32 formats, while the conversion commands
handle NIP-19's `npub` and `nsec` formats.

## Integration test with rage

With Go and `rage` installed, run the encryption/decryption round-trip test:

```sh
./scripts/test-rage.sh
```

Set `RAGE` to use a `rage` executable at a nonstandard path.

## Attach an age file to a Nostr note

`nostr-age-note` supports two ways to carry an encrypted `.age` file in a Nostr
note. By default it stores the binary bytes in a private PNG chunk, with a
small, highly compressible generated cover:

```sh
go install ./cmd/nostr-age-note
nostr-age-note embed secret.txt.age
```

It prints the timestamped filename. To use an existing PNG and choose the
output path:

```sh
nostr-age-note embed secret.txt.age --cover cover.png -o cover-with-age.png
```

An alternative `vs` format follows the byte-to-variation-selector mapping
described in [Paul Butler's article](https://paulbutler.org/2025/smuggling-arbitrary-data-through-an-emoji/):
each byte becomes one selector after a package emoji. It can be copied as text
and extracted directly, without an image host, but expands the payload to about
four UTF-8 bytes per input byte and invisible characters may be stripped by
software. Use it when text-carried data is useful:

```sh
nostr-age-note embed secret.txt.age --format vs
nostr-age-note extract nostr-age-YYYYMMDD-HHMMSS.txt
```

Upload the resulting PNG with a Nostr media uploader and put its URL in the
note. After downloading the original PNG, recover the encrypted file. Omitting
`-o` also generates a timestamped filename:

```sh
nostr-age-note extract cover-with-age.png
```

The PNG carrier adds approximately the `.age` file size plus a small cover,
without Base64's expansion. The PNG chunk is application-specific, so image services or editors
may discard it when transforming the image. Share the original PNG as a file
and extract the payload before opening or re-encoding it in an image editor.
