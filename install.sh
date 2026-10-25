#!/usr/bin/env bash
# Install tsp from GitHub releases.
#
#   curl -fsSL https://raw.githubusercontent.com/shizengzhou/tsp/main/install.sh | bash
#
# Environment:
#   TSP_VERSION       release tag to install (default: latest)
#   TSP_INSTALL_DIR   directory for the binary (default: ~/.local/bin)
#   TSP_BASE_URL      override the download base URL (for mirrors/tests)

set -eu

REPO_SLUG="shizengzhou/tsp"
BASE_URL="${TSP_BASE_URL:-https://github.com/${REPO_SLUG}}"
VERSION="${TSP_VERSION:-}"
INSTALL_DIR="${TSP_INSTALL_DIR:-$HOME/.local/bin}"

usage() {
	cat <<'EOF'
Install tsp from GitHub releases.

Usage:
  install.sh [--version TAG] [--dir DIR]

Options:
  --version TAG   Release tag to install, e.g. v0.1.0 (default: latest)
  --dir DIR       Where to put the tsp binary (default: ~/.local/bin)
  -h, --help      Show this help

Environment variables:
  TSP_VERSION      Same as --version
  TSP_INSTALL_DIR  Same as --dir
  TSP_BASE_URL     Override the download base URL
EOF
}

info() {
	printf '  %s\n' "$*"
}

fail() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

while [ $# -gt 0 ]; do
	case "$1" in
	--version)
		[ $# -ge 2 ] || fail "--version requires a value"
		VERSION="$2"
		shift 2
		;;
	--version=*)
		VERSION="${1#*=}"
		shift
		;;
	--dir)
		[ $# -ge 2 ] || fail "--dir requires a value"
		INSTALL_DIR="$2"
		shift 2
		;;
	--dir=*)
		INSTALL_DIR="${1#*=}"
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		usage >&2
		fail "unknown argument: $1"
		;;
	esac
done

command -v curl >/dev/null 2>&1 || fail "curl is required"
command -v tar >/dev/null 2>&1 || fail "tar is required"

case "$(uname -s)" in
Darwin) os="darwin" ;;
Linux) os="linux" ;;
*) fail "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) arch="amd64" ;;
arm64 | aarch64) arch="arm64" ;;
*) fail "unsupported architecture: $(uname -m)" ;;
esac

if [ -z "$VERSION" ]; then
	info "resolving the latest release…"
	latest_url="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "$BASE_URL/releases/latest")" ||
		fail "could not resolve the latest release; pass --version"
	VERSION="${latest_url##*/}"
	[ "$VERSION" != "latest" ] || fail "no releases found; pass --version"
fi

case "$VERSION" in
v*) TAG="$VERSION" ;;
*) TAG="v$VERSION" ;;
esac
NUM="${TAG#v}"

archive="tsp_${NUM}_${os}_${arch}.tar.gz"
download_url="$BASE_URL/releases/download/$TAG/$archive"
checksums_url="$BASE_URL/releases/download/$TAG/checksums.txt"

tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

info "downloading $archive ($os/$arch)"
curl -fsSL "$download_url" -o "$tmp_dir/$archive" ||
	fail "download failed: $download_url"

if curl -fsSL "$checksums_url" -o "$tmp_dir/checksums.txt" 2>/dev/null; then
	expected="$(awk -v name="$archive" '$2 == name { print $1 }' "$tmp_dir/checksums.txt")"
	if [ -n "$expected" ]; then
		actual=""
		if command -v sha256sum >/dev/null 2>&1; then
			actual="$(sha256sum "$tmp_dir/$archive" | awk '{ print $1 }')"
		elif command -v shasum >/dev/null 2>&1; then
			actual="$(shasum -a 256 "$tmp_dir/$archive" | awk '{ print $1 }')"
		fi
		if [ -z "$actual" ]; then
			info "warning: no sha256 tool found, skipping checksum verification"
		elif [ "$actual" != "$expected" ]; then
			fail "checksum mismatch for $archive"
		else
			info "checksum verified"
		fi
	else
		info "warning: $archive is not listed in checksums.txt"
	fi
else
	info "warning: checksums.txt unavailable, skipping verification"
fi

tar -xzf "$tmp_dir/$archive" -C "$tmp_dir" || fail "failed to extract $archive"
[ -f "$tmp_dir/tsp" ] || fail "archive does not contain a tsp binary"

mkdir -p "$INSTALL_DIR" || fail "could not create $INSTALL_DIR"
target="$INSTALL_DIR/tsp"
staged="$INSTALL_DIR/.tsp.tmp.$$"
cp "$tmp_dir/tsp" "$staged" || fail "could not write to $INSTALL_DIR"
chmod 0755 "$staged"
mv -f "$staged" "$target" || {
	rm -f "$staged"
	fail "could not install to $target"
}

info "installed tsp $NUM to $target"

case ":$PATH:" in
*":$INSTALL_DIR:"*) ;;
*)
	printf '\n%s is not on your PATH. Add it with:\n  export PATH="%s:$PATH"\n' "$INSTALL_DIR" "$INSTALL_DIR"
	;;
esac

printf '\nRun "tsp omz themes" to browse your oh-my-zsh themes.\n'
