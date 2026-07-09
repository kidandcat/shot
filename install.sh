#!/bin/sh
# Install the shot binary and, optionally, the Claude Code skill.
#
#   ./install.sh              build + install to ~/.local/bin, install the skill
#   ./install.sh --no-skill   binary only
#
# Requires Go 1.22+.

set -eu

BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
SKILL_DIR="${SKILL_DIR:-$HOME/.claude/skills}"
INSTALL_SKILL=1

for arg in "$@"; do
	case "$arg" in
	--no-skill) INSTALL_SKILL=0 ;;
	-h | --help)
		sed -n '2,8p' "$0" | sed 's/^# \{0,1\}//'
		exit 0
		;;
	*)
		echo "unknown option: $arg" >&2
		exit 1
		;;
	esac
done

command -v go >/dev/null 2>&1 || {
	echo "shot: Go is required (https://go.dev/dl/)" >&2
	exit 1
}

src="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"

echo "building..."
(cd "$src" && go build -o shot .)

mkdir -p "$BIN_DIR"
install -m 0755 "$src/shot" "$BIN_DIR/shot"
echo "installed $BIN_DIR/shot"

case ":$PATH:" in
*":$BIN_DIR:"*) ;;
*) echo "warning: $BIN_DIR is not on your PATH" >&2 ;;
esac

if [ "$INSTALL_SKILL" -eq 1 ]; then
	# Symlink rather than copy, so `git pull` keeps the skill current.
	mkdir -p "$SKILL_DIR"
	rm -rf "$SKILL_DIR/shot"
	ln -s "$src/skills/shot" "$SKILL_DIR/shot"
	echo "installed skill  $SKILL_DIR/shot -> $src/skills/shot"
fi

echo
echo "try it:  shot some-screenshot.png --open"
