#!/usr/bin/env bash
# Removes ~/.local/bin/mcm. Leaves your servers and config alone.
set -euo pipefail

dest="$HOME/.local/bin/mcm"

if [ ! -e "$dest" ]; then
  echo "mcm is not installed at $dest"
  exit 0
fi

# Stop a backgrounded `mcm web start` first, so it isn't left running
# from a binary that no longer exists.
"$dest" web stop >/dev/null 2>&1 || true

rm -f "$dest"
echo "removed $dest"
echo "your servers and config were left in place (see README.md → Configuration)"
