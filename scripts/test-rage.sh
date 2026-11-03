#!/bin/sh
set -eu

rage_bin=${RAGE:-rage}
if ! command -v "$rage_bin" >/dev/null 2>&1; then
	printf 'rage executable not found: %s\n' "$rage_bin" >&2
	exit 1
fi

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM
umask 077

cd "$repo_root"
go build -o "$tmp_dir/age-plugin-nostr" .
key_output=$("$tmp_dir/age-plugin-nostr" keygen)
recipient=$(printf '%s\n' "$key_output" | sed -n 's/^age recipient: //p')
nsec=$(printf '%s\n' "$key_output" | sed -n 's/^Nostr secret key: //p')
if [ -z "$recipient" ] || [ -z "$nsec" ]; then
	printf 'keygen did not return age recipient and Nostr secret key\n' >&2
	exit 1
fi

printf '%s\n' "$nsec" > "$tmp_dir/nsec.txt"
"$tmp_dir/age-plugin-nostr" identity-file "$tmp_dir/nsec.txt" > "$tmp_dir/identity.txt"
printf 'rage plugin integration test\n' > "$tmp_dir/plaintext.txt"
PATH="$tmp_dir:$PATH" "$rage_bin" -r "$recipient" -o "$tmp_dir/ciphertext.age" "$tmp_dir/plaintext.txt"
PATH="$tmp_dir:$PATH" "$rage_bin" -d -i "$tmp_dir/identity.txt" -o "$tmp_dir/decrypted.txt" "$tmp_dir/ciphertext.age"
cmp "$tmp_dir/plaintext.txt" "$tmp_dir/decrypted.txt"
printf 'rage Nostr plugin integration test passed\n'
