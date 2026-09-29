#!/bin/sh
# 编译各平台单文件可执行程序（webrpc 动态库从 sdk/ 嵌入，dist 无需再附带 .so/.dylib/.dll）
set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
SDK_DIR="${ROOT}/sdk"
DIST_DIR="${ROOT}/dist"
RUNTIME_DIR="${ROOT}/internal/webrpc/runtimelib"

cd "$ROOT"

log() { printf '==> %s\n' "$*"; }
warn() { printf 'warning: %s\n' "$*" >&2; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

command -v go >/dev/null 2>&1 || die "未找到 go，请先安装: https://go.dev/dl/"

need_file() { [ -f "$1" ] || die "缺少 SDK 文件: $1"; }

prepare_embed() {
  os=$1
  src=$2
  name=$3
  dest="${RUNTIME_DIR}/${os}/${name}"
  need_file "$src"
  mkdir -p "$(dirname "$dest")"
  cp "$src" "$dest"
  chmod 0755 "$dest"
}

go_build() {
  goos=$1
  goarch=$2
  outfile=$3
  cgo=$4
  export GOOS="$goos"
  export GOARCH="$goarch"
  export CGO_ENABLED="$cgo"
  unset CC
  go build -trimpath -ldflags="-s -w" -o "$outfile" .
}

build_darwin_arm64() {
  prepare_embed darwin_arm64 "${SDK_DIR}/libwebrpc-Mac.dylib" libwebrpc.dylib
  out="${DIST_DIR}/darwin-arm64/p2pssh-server"
  mkdir -p "${DIST_DIR}/darwin-arm64"
  log "build darwin/arm64 (cgo=1, shell PTY)"
  # Native mac build: enable cgo so in-app shell PTY works.
  if [ "$(uname -s)" = "Darwin" ] && [ "$(uname -m)" = "arm64" ]; then
    go_build darwin arm64 "$out" 1
  else
    warn "非本机 darwin/arm64，将以 cgo=0 编译（无 shell PTY，仅 tcp 转发）"
    go_build darwin arm64 "$out" 0
  fi
}

build_linux() {
  arch=$1
  case $arch in
    amd64)
      prepare_embed linux_amd64 "${SDK_DIR}/libwebrpc-Linux.so" libwebrpc.so
      ;;
    arm64)
      prepare_embed linux_arm64 "${SDK_DIR}/libwebrpc-Linux-arm64.so" libwebrpc.so
      ;;
    *)
      return 1
      ;;
  esac
  outdir="${DIST_DIR}/linux-${arch}"
  mkdir -p "$outdir"
  # Prefer native linux+cgo when possible; cross builds stay cgo=0 (tcp-only).
  cgo=0
  if [ "$(uname -s)" = "Linux" ]; then
    host_arch=$(uname -m)
    if { [ "$arch" = "amd64" ] && [ "$host_arch" = "x86_64" ]; } || \
       { [ "$arch" = "arm64" ] && [ "$host_arch" = "aarch64" ]; }; then
      cgo=1
    fi
  fi
  log "build linux/${arch} (cgo=${cgo})"
  go_build linux "$arch" "${outdir}/p2pssh-server" "$cgo"
}

build_windows_amd64() {
  prepare_embed windows_amd64 "${SDK_DIR}/webrpc-Windows.dll" libwebrpc.dll
  out="${DIST_DIR}/windows-amd64/p2pssh-server.exe"
  mkdir -p "${DIST_DIR}/windows-amd64"
  log "build windows/amd64 (tcp only)"
  go_build windows amd64 "$out" 0
}

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

BUILT=0
FAILED=0
try_build() {
  if "$@"; then
    BUILT=$((BUILT + 1))
  else
    FAILED=$((FAILED + 1))
  fi
}

try_build build_darwin_arm64
try_build build_linux amd64
try_build build_linux arm64
try_build build_windows_amd64

log "完成: ${BUILT} 个可执行文件 -> ${DIST_DIR}"
[ "$BUILT" -eq 0 ] && die "没有成功编译任何平台"
[ "$FAILED" -gt 0 ] && warn "${FAILED} 个目标未编译"

find "$DIST_DIR" -type f \( -name 'p2pssh-server' -o -name 'p2pssh-server.exe' \) | sort | sed 's/^/  /'
exit 0
