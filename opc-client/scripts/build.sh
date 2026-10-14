#!/usr/bin/env bash
set -euo pipefail
client_root="$(cd "$(dirname "$0")/.." && pwd)"
platform="${1:-darwin/arm64}"
wails_cli="${WAILS_BIN:-}"
if [[ -z "$wails_cli" ]]; then
  wails_cli="$(command -v wails || true)"
fi
if [[ -z "$wails_cli" && -x "$client_root/../.desktop-tools/bin/wails" ]]; then
  wails_cli="$client_root/../.desktop-tools/bin/wails"
fi
if [[ -z "$wails_cli" ]]; then echo 'Install Wails 2.12.0 or set WAILS_BIN.' >&2; exit 1; fi
cd "$client_root/frontend"
npm ci --no-audit --no-fund
npm run build
python3 "$client_root/scripts/embed-assets.py"
cd "$client_root/desktop"
args=(-platform "$platform" -s -trimpath -nosyncgomod)
if [[ "$platform" == windows/* ]]; then args+=(-nsis); fi
"$wails_cli" build "${args[@]}"
