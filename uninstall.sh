#!/usr/bin/env bash
# Uninstall tsp.
#
#   curl -fsSL https://raw.githubusercontent.com/shizengzhou/tsp/main/uninstall.sh | bash
#
# Environment:
#   TSP_INSTALL_DIR   directory that holds the binary (default: ~/.local/bin)

set -eu

INSTALL_DIR="${TSP_INSTALL_DIR:-$HOME/.local/bin}"

usage() {
	cat <<'EOF'
Uninstall tsp.

Usage:
  uninstall.sh [--dir DIR]

Options:
  --dir DIR       Where the tsp binary was installed (default: ~/.local/bin)
  -h, --help      Show this help

Environment variables:
  TSP_INSTALL_DIR  Same as --dir

The install directory is removed only once it is empty after the binary is
deleted. Shell profiles and configuration that tsp wrote are left untouched.
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

target="$INSTALL_DIR/tsp"
if [ ! -f "$target" ]; then
	info "tsp is not installed in $INSTALL_DIR"
	exit 0
fi

rm -f "$target" || fail "could not remove $target"
info "removed $target"

# Keep the directory when other tools live in it.
if [ -z "$(ls -A "$INSTALL_DIR" 2>/dev/null)" ]; then
	rmdir "$INSTALL_DIR" 2>/dev/null || fail "could not remove empty $INSTALL_DIR"
	info "removed $INSTALL_DIR"
	printf '\nIf your shell profile still adds %s to PATH, remove that line.\n' "$INSTALL_DIR"
fi
