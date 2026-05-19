#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

APP_NAME=${APP_NAME:-jlpt-desktop}
VERSION=${VERSION:-$(awk -F'"' '/^[[:space:]]{2}version:[[:space:]]*"/ {print $2; exit}' build/config.yml)}
GOOS_NAME=${GOOS_NAME:-$(go env GOOS)}
GOARCH_NAME=${GOARCH_NAME:-$(go env GOARCH)}
RELEASE_DIR=${RELEASE_DIR:-bin/release}
ARTIFACT_BASE="${APP_NAME}_${VERSION}_${GOOS_NAME}_${GOARCH_NAME}"
STAGING="${RELEASE_DIR}/${ARTIFACT_BASE}"

GO_BIN=$(go env GOBIN)
GO_PATH_BIN="$(go env GOPATH)/bin"
if [[ -n "$GO_BIN" ]]; then
  export PATH="${PATH}:${GO_BIN}:${GO_PATH_BIN}"
else
  export PATH="${PATH}:${GO_PATH_BIN}"
fi

if ! command -v wails3 >/dev/null 2>&1; then
  echo "wails3 is required. Install the pinned CLI with:" >&2
  echo "  go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.93" >&2
  exit 1
fi

wails3 task build

go test ./... -run TestReleaseSmokeImportRunAndStudy -count=1

rm -rf "$STAGING"
mkdir -p "$STAGING"
cp "bin/${APP_NAME}" "$STAGING/${APP_NAME}"
cp build/linux/${APP_NAME}.desktop "$STAGING/${APP_NAME}.desktop"
cp build/generated/appicon.png "$STAGING/${APP_NAME}.png"
cp ../docs/desktop-release.md "$STAGING/INSTALL_AND_MIGRATION.md"

cat > "$STAGING/install-linux-user.sh" <<'INSTALL'
#!/usr/bin/env bash
set -euo pipefail
PREFIX=${PREFIX:-"$HOME/.local"}
APP_NAME=jlpt-desktop
install -d "$PREFIX/bin" "$PREFIX/share/applications" "$PREFIX/share/icons/hicolor/1024x1024/apps"
install -m 0755 "$APP_NAME" "$PREFIX/bin/$APP_NAME"
install -m 0644 "$APP_NAME.desktop" "$PREFIX/share/applications/$APP_NAME.desktop"
install -m 0644 "$APP_NAME.png" "$PREFIX/share/icons/hicolor/1024x1024/apps/$APP_NAME.png"
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$PREFIX/share/applications" || true
fi
echo "Installed $APP_NAME to $PREFIX/bin/$APP_NAME"
INSTALL
chmod +x "$STAGING/install-linux-user.sh"

(
  cd "$RELEASE_DIR"
  tar -czf "${ARTIFACT_BASE}.tar.gz" "$ARTIFACT_BASE"
  sha256sum "${ARTIFACT_BASE}.tar.gz" > "${ARTIFACT_BASE}.tar.gz.sha256"
)

echo "Release artifact: ${RELEASE_DIR}/${ARTIFACT_BASE}.tar.gz"
echo "Checksum: ${RELEASE_DIR}/${ARTIFACT_BASE}.tar.gz.sha256"
