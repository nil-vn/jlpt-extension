#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

ARTIFACT=${1:-}
if [[ -z "$ARTIFACT" ]]; then
  ARTIFACT=$(find bin/release -maxdepth 1 -name 'jlpt-desktop_*.tar.gz' -type f | sort | tail -n 1 || true)
fi
if [[ -z "$ARTIFACT" || ! -f "$ARTIFACT" ]]; then
  echo "Usage: $0 path/to/jlpt-desktop_<version>_<os>_<arch>.tar.gz" >&2
  exit 1
fi

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

tar -xzf "$ARTIFACT" -C "$TMP_DIR"
APP_DIR=$(find "$TMP_DIR" -mindepth 1 -maxdepth 1 -type d | head -n 1)
[[ -x "$APP_DIR/jlpt-desktop" ]]
[[ -f "$APP_DIR/jlpt-desktop.desktop" ]]
[[ -f "$APP_DIR/INSTALL_AND_MIGRATION.md" ]]

(cd "$APP_DIR" && PREFIX="$TMP_DIR/prefix" ./install-linux-user.sh)
[[ -x "$TMP_DIR/prefix/bin/jlpt-desktop" ]]

ldd "$TMP_DIR/prefix/bin/jlpt-desktop" >/dev/null

go test ./... -run TestReleaseSmokeImportRunAndStudy -count=1

echo "Smoke OK: extracted, user-installed, linked, and imported sample JSON via desktop service."
echo "GUI launch smoke: run '$TMP_DIR/prefix/bin/jlpt-desktop' in a graphical Linux session."
