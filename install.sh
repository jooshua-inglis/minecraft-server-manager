#!/usr/bin/env bash
# Installs the prebuilt mcm binary from a GitHub release to ~/.local/bin/mcm.
# Standalone (no checkout needed), so it can be piped:
#
#   curl -fsSL https://raw.githubusercontent.com/jooshua-inglis/minecraft-server-manager/main/install.sh | bash
#   curl -fsSL .../install.sh | MCM_VERSION=v0.1.0 bash      # pin a version
#
# MCM_VERSION defaults to "latest" (the rolling build of main). Linux amd64
# and arm64 only. MCM_DOWNLOAD_BASE overrides where release assets are
# fetched from (a mirror, or a local server for testing).
set -euo pipefail

repo="jooshua-inglis/minecraft-server-manager"
version="${MCM_VERSION:-latest}"
base="${MCM_DOWNLOAD_BASE:-https://github.com/$repo/releases/download/$version}"
bin_dir="$HOME/.local/bin"
dest="$bin_dir/mcm"

die() { echo "error: $*" >&2; exit 1; }

[ "$(uname -s)" = "Linux" ] || die "prebuilt binaries are Linux only (got $(uname -s))"
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) die "unsupported architecture: $(uname -m) (need x86_64 or aarch64)" ;;
esac
asset="mcm-linux-$arch"

# fetch URL [FILE]: write to FILE, or stdout if omitted.
fetch() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL ${2:+-o "$2"} "$1"
  elif command -v wget >/dev/null 2>&1; then
    wget -q -O "${2:--}" "$1"
  else
    die "need curl or wget"
  fi
}
command -v sha256sum >/dev/null 2>&1 || die "need sha256sum to verify the download"

mkdir -p "$bin_dir"
# Download beside the destination and rename into place, so reinstalling
# works even while `mcm web` is running from the old binary.
tmp="$(mktemp "$bin_dir/.mcm.XXXXXX")"
trap 'rm -f "$tmp"' EXIT

echo "downloading $asset ($version)..."
fetch "$base/$asset" "$tmp" ||
  die "couldn't download $base/$asset (has a release been published yet?)"

want="$(fetch "$base/SHA256SUMS" | awk -v f="$asset" '$2 == f { print $1 }')" ||
  die "couldn't download $base/SHA256SUMS"
[ -n "$want" ] || die "$asset isn't listed in SHA256SUMS"
got="$(sha256sum "$tmp" | awk '{ print $1 }')"
[ "$got" = "$want" ] || die "checksum mismatch for $asset (expected $want, got $got)"

chmod 0755 "$tmp"
mv -f "$tmp" "$dest"

echo "installed $dest ($("$dest" version))"

case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *) echo "note: $bin_dir is not on your PATH — add it to your shell profile" >&2 ;;
esac
