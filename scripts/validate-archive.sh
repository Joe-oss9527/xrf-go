#!/usr/bin/env bash

set -euo pipefail

# Validate a release tar.gz contains the correct binary and architecture
# Usage: scripts/validate-archive.sh <archive_path> <expected_arch>
#   expected_arch: amd64 | arm64

archive_path="${1:-}"
expected_arch="${2:-}"

if [[ -z "$archive_path" || -z "$expected_arch" ]]; then
  echo "Usage: $0 <archive_path> <expected_arch>" >&2
  exit 2
fi

if [[ ! -f "$archive_path" ]]; then
  echo "Archive not found: $archive_path" >&2
  exit 2
fi

if ! command -v tar >/dev/null 2>&1; then
  echo "tar command not found" >&2
  exit 2
fi

if ! command -v file >/dev/null 2>&1; then
  echo "file command not found; cannot verify architecture" >&2
  exit 2
fi

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

# List contents
contents=$(tar -tzf "$archive_path") || {
  echo "Failed to list archive contents: $archive_path" >&2
  exit 1
}

# Expect a single top-level file named xrf-linux-<arch>
expected_name="xrf-linux-${expected_arch}"
if ! echo "$contents" | grep -qx "$expected_name"; then
  echo "Archive contents unexpected. Expected only: $expected_name" >&2
  echo "Actual contents:" >&2
  echo "$contents" >&2
  exit 1
fi

tar -xzf "$archive_path" -C "$tmpdir"
bin_path="$tmpdir/$expected_name"

if [[ ! -f "$bin_path" ]]; then
  echo "Extracted binary missing: $bin_path" >&2
  exit 1
fi

chmod +x "$bin_path" || true
desc=$(file -b "$bin_path" || true)

case "$expected_arch" in
  amd64)
    if ! echo "$desc" | grep -Eiq 'ELF 64-bit.*x86-64|ELF 64-bit.*AMD64'; then
      echo "Architecture mismatch for $archive_path. Expected amd64, got: $desc" >&2
      exit 1
    fi
    ;;
  arm64)
    if ! echo "$desc" | grep -Eiq 'ELF 64-bit.*aarch64|ELF 64-bit.*ARM aarch64|ELF 64-bit.*ARM64'; then
      echo "Architecture mismatch for $archive_path. Expected arm64, got: $desc" >&2
      exit 1
    fi
    ;;
  *)
    echo "Unknown expected_arch: $expected_arch" >&2
    exit 2
    ;;
esac

echo "Validated: $archive_path -> $desc"
