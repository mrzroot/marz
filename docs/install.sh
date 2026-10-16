#!/bin/sh
# marz one-line installer (macOS, Linux, FreeBSD, WSL):
#   curl -fsSL https://mrzroot.github.io/marz/install.sh | sh
# Downloads the right binary from the latest GitHub release, resumes if the connection drops,
# verifies its SHA-256 and installs it to ~/.local/bin (or $MARZ_BIN_DIR). No root needed.
# Options: MARZ_VERSION=v0.1.0  MARZ_BIN_DIR=/usr/local/bin  MARZ_BASE_URL=<mirror of the release assets>
set -eu

say() { printf '%s\n' "$*" >&2; }
die() { say "marz install: $*"; exit 1; }

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux) os=linux ;; darwin) os=darwin ;; freebsd) os=freebsd ;;
  msys*|mingw*|cygwin*) die "on Windows use PowerShell:  irm https://mrzroot.github.io/marz/install.ps1 | iex" ;;
  *) die "unsupported OS $os" ;;
esac
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;; aarch64|arm64) arch=arm64 ;; armv7*|armv6*|armhf) arch=arm ;; i?86) arch=386 ;;
  *) die "unsupported CPU $arch" ;;
esac
asset="marz_${os}_${arch}"

if [ -n "${MARZ_BASE_URL:-}" ]; then base="$MARZ_BASE_URL"
elif [ -n "${MARZ_VERSION:-}" ]; then base="https://github.com/mrzroot/marz/releases/download/$MARZ_VERSION"
else base="https://github.com/mrzroot/marz/releases/latest/download"; fi

bindir="${MARZ_BIN_DIR:-$HOME/.local/bin}"
mkdir -p "$bindir" || die "cannot create $bindir"
tmp=$(mktemp -d 2>/dev/null || mktemp -d -t marz)
trap 'rm -rf "$tmp"' EXIT INT TERM

# fetch URL FILE: resumable, retried; resumes on flaky links
fetch() {
  i=0
  while :; do
    i=$((i + 1))
    if command -v curl >/dev/null 2>&1; then
      curl -fL --retry 3 --connect-timeout 20 -sS -C - -o "$2" "$1" && return 0
    elif command -v wget >/dev/null 2>&1; then
      wget -q -c -O "$2" "$1" && return 0
    else
      die "need curl or wget"
    fi
    [ "$i" -ge 15 ] && return 1
    say "  connection dropped; resuming ($i)..."
    sleep 2
  done
}

say "==> Downloading $asset"
fetch "$base/$asset" "$tmp/marz" || die "download failed: $base/$asset"
fetch "$base/SHA256SUMS" "$tmp/SHA256SUMS" || die "could not download SHA256SUMS"
want=$(grep " $asset\$" "$tmp/SHA256SUMS" | cut -d' ' -f1)
[ -n "$want" ] || die "$asset is not listed in SHA256SUMS"
if command -v sha256sum >/dev/null 2>&1; then got=$(sha256sum "$tmp/marz" | cut -d' ' -f1)
else got=$(shasum -a 256 "$tmp/marz" | cut -d' ' -f1); fi
[ "$want" = "$got" ] || die "checksum mismatch (expected $want, got $got)"
say "  checksum ok"
chmod +x "$tmp/marz"
mv -f "$tmp/marz" "$bindir/marz"
say "==> Installed $("$bindir/marz" version) to $bindir/marz"

case ":$PATH:" in
  *":$bindir:"*) ;;
  *)
    line="export PATH=\"$bindir:\$PATH\""
    for rc in "$HOME/.profile" "$HOME/.bashrc" "$HOME/.zshrc"; do
      if [ -f "$rc" ] || [ "$rc" = "$HOME/.profile" ]; then
        grep -qs "$bindir" "$rc" || printf '\n# added by the marz installer\n%s\n' "$line" >> "$rc"
      fi
    done
    say "  added $bindir to PATH in your shell profile; open a new terminal or run:  $line"
    ;;
esac
say ""
say "Try:  cd your-project && marz"
say "      marz probe fonts.googleapis.com api.openai.com"
