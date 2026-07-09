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

# When piped from curl, "$0" is the shell itself, not a path — never fall back to
# the current directory, which might be any unrelated Go module.
src=""
if [ -f "$0" ]; then
	src="$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)"
fi

# No checkout next to the script, so fetch one.
if [ -z "$src" ] || [ ! -f "$src/go.mod" ]; then
	command -v git >/dev/null 2>&1 || {
		echo "shot: git is required" >&2
		exit 1
	}
	src="$(mktemp -d)"
	EPHEMERAL=1
	trap 'rm -rf "$src"' EXIT INT TERM
	echo "fetching shot..."
	git clone --depth 1 -q https://github.com/kidandcat/shot.git "$src"
fi

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
	mkdir -p "$SKILL_DIR"
	rm -rf "$SKILL_DIR/shot"
	if [ "${EPHEMERAL:-0}" -eq 1 ]; then
		# The checkout is a temp dir we are about to delete; a symlink would dangle.
		cp -R "$src/skills/shot" "$SKILL_DIR/shot"
		echo "installed skill  $SKILL_DIR/shot"
	else
		# Symlink from a real checkout, so `git pull` keeps the skill current.
		ln -s "$src/skills/shot" "$SKILL_DIR/shot"
		echo "installed skill  $SKILL_DIR/shot -> $src/skills/shot"
	fi
fi

echo
echo "try it:  shot some-screenshot.png --open"
