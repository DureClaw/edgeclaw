#!/usr/bin/env sh
# edgeclaw installer — detects your OS/CPU and fetches the matching prebuilt binary.
#   curl -fsSL https://github.com/DureClaw/edgeclaw/releases/latest/download/install.sh | sh
set -e
REPO="DureClaw/edgeclaw"
os=$(uname -s); arch=$(uname -m)
case "$os" in
  Linux)  goos=linux ;;
  Darwin) goos=darwin ;;
  MINGW*|MSYS*|CYGWIN*) goos=windows ;;
  *) echo "unsupported OS: $os"; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64) goarch=amd64 ;;
  arm64|aarch64) goarch=arm64 ;;
  armv7l) goarch=armv7 ;;
  armv6l) goarch=armv6 ;;
  riscv64) goarch=riscv64 ;;
  loongarch64|loong64) goarch=loong64 ;;
  mips64) goarch=mips64le ;;
  *) echo "unsupported arch: $arch"; exit 1 ;;
esac
ext=""; [ "$goos" = windows ] && ext=".exe"
asset="edgeclaw-${goos}-${goarch}${ext}"
url="https://github.com/${REPO}/releases/latest/download/${asset}"
echo "edgeclaw: downloading ${asset} ..."
curl -fsSL "$url" -o edgeclaw${ext}
chmod +x edgeclaw${ext} 2>/dev/null || true
echo "✓ saved ./edgeclaw${ext}"
echo "  run: STATE_SERVER=<bus:4000> OAH_SECRET=<token> WORK_KEY=<WK> ./edgeclaw${ext}"
