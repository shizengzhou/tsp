#!/usr/bin/env bash
# Verifies install.sh against a local fake release server: one install that
# succeeds, and the three ways checksum verification must refuse to install.
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
work=$(mktemp -d)
server_pid=""

cleanup() {
	if [ -n "$server_pid" ]; then
		kill "$server_pid" 2>/dev/null || true
		wait "$server_pid" 2>/dev/null || true
	fi
	rm -rf "$work"
}
trap cleanup EXIT

version=0.0.1
os=$(uname -s | tr 'A-Z' 'a-z')
case "$os" in
darwin | linux) ;;
*)
	echo "unsupported OS for this check: $os" >&2
	exit 1
	;;
esac
arch=$(uname -m)
case "$arch" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*)
	echo "unsupported architecture for this check: $arch" >&2
	exit 1
	;;
esac

archive_name="tsp_${version}_${os}_${arch}.tar.gz"
release="$work/server/releases/download/v$version"
mkdir -p "$release" "$work/pkg"
printf '#!/bin/sh\necho fake tsp\n' >"$work/pkg/tsp"
chmod +x "$work/pkg/tsp"
tar -czf "$release/$archive_name" -C "$work/pkg" tsp

if command -v sha256sum >/dev/null 2>&1; then
	sum=$(sha256sum "$release/$archive_name" | awk '{ print $1 }')
else
	sum=$(shasum -a 256 "$release/$archive_name" | awk '{ print $1 }')
fi
printf '%s  %s\n' "$sum" "$archive_name" >"$release/checksums.txt"

port=$(python3 -c 'import socket; s = socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')
python3 -m http.server "$port" --directory "$work/server" >/dev/null 2>&1 &
server_pid=$!
sleep 1
base="http://127.0.0.1:$port"

install() {
	local dir=$1
	TSP_BASE_URL="$base" TSP_VERSION="$version" TSP_INSTALL_DIR="$dir" \
		bash "$repo_root/install.sh"
}

expect_refusal() {
	local label=$1 want=$2 dir=$3 output
	if output=$(install "$dir" 2>&1); then
		echo "FAIL: $label installed anyway" >&2
	exit 1
	fi
	if ! printf '%s' "$output" | grep -qF "$want"; then
		echo "FAIL: $label did not mention \"$want\"" >&2
		printf '%s\n' "$output" >&2
		exit 1
	fi
	if [ -e "$dir/tsp" ]; then
		echo "FAIL: $label left a binary behind" >&2
		exit 1
	fi
	echo "ok: $label"
}

echo "=== checksums present ==="
install "$work/bin-ok"
[ -x "$work/bin-ok/tsp" ] || {
	echo "FAIL: tsp was not installed" >&2
	exit 1
}
echo "ok: installed"

echo "=== checksums missing ==="
rm -f "$release/checksums.txt"
expect_refusal "missing checksums" "checksums.txt" "$work/bin-missing"

echo "=== archive not listed ==="
printf '%s  other.tar.gz\n' "$sum" >"$release/checksums.txt"
expect_refusal "unlisted archive" "not listed" "$work/bin-unlisted"

echo "=== checksum mismatch ==="
printf '0000000000000000000000000000000000000000000000000000000000000000  %s\n' \
	"$archive_name" >"$release/checksums.txt"
expect_refusal "checksum mismatch" "checksum mismatch" "$work/bin-mismatch"

echo "all install.sh checks passed"
