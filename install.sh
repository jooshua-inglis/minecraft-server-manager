#!/usr/bin/env bash
# Builds mcm from this checkout and installs it to ~/.local/bin/mcm.
# Needs Go; the web dashboard is prebuilt and embedded, so no JS toolchain.
set -euo pipefail

bin_dir="$HOME/.local/bin"
dest="$bin_dir/mcm"

if ! command -v go >/dev/null 2>&1; then
  echo "error: Go is required to build mcm (https://go.dev/dl/)" >&2
  exit 1
fi

cd "$(dirname "$0")"
mkdir -p "$bin_dir"

# Build beside the destination and rename into place, so reinstalling
# works even while `mcm web` is running from the old binary.
tmp="$(mktemp "$bin_dir/.mcm.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
go build -o "$tmp" ./cmd/mcm
chmod 0755 "$tmp"
mv -f "$tmp" "$dest"

echo "installed $dest"

case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *) echo "note: $bin_dir is not on your PATH — add it to your shell profile" >&2 ;;
esac
