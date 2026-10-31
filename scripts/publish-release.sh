#!/usr/bin/env bash
# 将 dist/ 里已编译好的各平台二进制打包并发布到 GitHub Releases。
#
# ========== 配置（改版本号主要改这里）==========
# 也可临时覆盖：VERSION=0.2.0 ./scripts/publish-release.sh
: "${VERSION:=1.2.0}"
# 是否发布为草稿（1=草稿，0=正式发布）
: "${DRAFT:=0}"
# 仓库（留空则用当前 git remote origin），例如 xiaoming-software/p2pssh-server
: "${REPO:=}"
# ============================================
#
# 用法：
#   ./scripts/publish-release.sh                 # 按上面 VERSION 发布
#   VERSION=0.2.0 ./scripts/publish-release.sh   # 临时覆盖版本号
#   ./scripts/publish-release.sh --dry-run        # 只打包，不上传
#   ./scripts/publish-release.sh --draft          # 先发草稿
#
# 前置：
#   1. 已执行 ./build.sh，产物在 dist/（四个平台齐全）
#   2. 已安装 gh：brew install gh && gh auth login

set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/dist"
ASSETS_DIR="${PROJECT_ROOT}/release-assets"
DRY_RUN="0"

log() { echo "[INFO] $*"; }
err() { echo "[ERROR] $*" >&2; }

usage() {
  cat <<EOF
用法: $0 [--dry-run] [--draft] [--help]

在脚本顶部修改 VERSION，或临时覆盖：
  VERSION=0.1.1 ./scripts/publish-release.sh
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN="1"; shift ;;
    --draft) DRAFT="1"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) err "未知参数: $1"; usage; exit 1 ;;
  esac
done

# 规范化版本：配置写 0.1.0，tag 用 v0.1.0
VERSION="${VERSION#v}"
if [[ ! "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][A-Za-z0-9.]+)?$ ]]; then
  err "VERSION 格式不正确: ${VERSION}（示例: 0.1.0）"
  exit 1
fi
TAG="v${VERSION}"
TITLE="p2pssh-server ${TAG}"

cd "${PROJECT_ROOT}"

resolve_repo() {
  if [[ -n "${REPO}" ]]; then
    echo "${REPO}"
    return
  fi
  local url
  url="$(git remote get-url origin 2>/dev/null || true)"
  if [[ -z "${url}" ]]; then
    err "无法解析 git remote origin，请设置脚本顶部 REPO=owner/name"
    exit 1
  fi
  # https://github.com/owner/repo.git 或 git@github.com:owner/repo.git
  echo "${url}" \
    | sed -E 's#^git@github\.com:#https://github.com/#' \
    | sed -E 's#\.git$##' \
    | sed -E 's#^https://github.com/##'
}

check_prereqs() {
  if [[ ! -d "${DIST_DIR}" ]]; then
    err "找不到 ${DIST_DIR}，请先 ./build.sh"
    exit 1
  fi
  if [[ "${DRY_RUN}" == "1" ]]; then
    return 0
  fi
  if ! command -v gh >/dev/null 2>&1; then
    err "未找到 gh。请先执行: brew install gh && gh auth login"
    exit 1
  fi
  if ! gh auth status >/dev/null 2>&1; then
    err "gh 未登录。请先执行: gh auth login"
    exit 1
  fi
}

# 把单个二进制打进 tar.gz；归档内文件名保持 p2pssh-server
pack_unix() {
  local src="$1"
  local out_tar="$2"
  local tmp
  tmp="$(mktemp -d "${TMPDIR:-/tmp}/p2pssh-pack.XXXXXX")"
  cp -f "${src}" "${tmp}/p2pssh-server"
  chmod 0755 "${tmp}/p2pssh-server"
  tar -C "${tmp}" -czf "${out_tar}" p2pssh-server
  rm -rf "${tmp}"
}

prepare_assets() {
  rm -rf "${ASSETS_DIR}"
  mkdir -p "${ASSETS_DIR}"

  local darwin_bin linux_amd64_bin linux_arm64_bin win_exe
  darwin_bin="${DIST_DIR}/darwin-arm64/p2pssh-server"
  linux_amd64_bin="${DIST_DIR}/linux-amd64/p2pssh-server"
  linux_arm64_bin="${DIST_DIR}/linux-arm64/p2pssh-server"
  win_exe="${DIST_DIR}/windows-amd64/p2pssh-server.exe"

  local missing=0
  if [[ ! -f "${darwin_bin}" ]]; then
    err "缺少 macOS 产物: ${darwin_bin}"
    missing=1
  fi
  if [[ ! -f "${linux_amd64_bin}" ]]; then
    err "缺少 Linux amd64 产物: ${linux_amd64_bin}"
    missing=1
  fi
  if [[ ! -f "${linux_arm64_bin}" ]]; then
    err "缺少 Linux arm64 产物: ${linux_arm64_bin}"
    missing=1
  fi
  if [[ ! -f "${win_exe}" ]]; then
    err "缺少 Windows 产物: ${win_exe}"
    missing=1
  fi
  if [[ "${missing}" -ne 0 ]]; then
    err "请先完整执行 ./build.sh，确保四个平台产物齐全"
    exit 1
  fi

  log "打包 Unix .tar.gz ..."
  pack_unix "${darwin_bin}" "${ASSETS_DIR}/p2pssh-server-darwin-arm64.tar.gz"
  pack_unix "${linux_amd64_bin}" "${ASSETS_DIR}/p2pssh-server-linux-amd64.tar.gz"
  pack_unix "${linux_arm64_bin}" "${ASSETS_DIR}/p2pssh-server-linux-arm64.tar.gz"
  cp -f "${win_exe}" "${ASSETS_DIR}/p2pssh-server-windows-amd64.exe"

  log "待上传文件："
  ls -lh "${ASSETS_DIR}"
}

release_notes() {
  cat <<EOF
## p2pssh-server ${TAG}

基于 webrpc 的 P2P 远程终端 / SSH 端口转发服务端。客户端请使用 [File2File](https://github.com/xiaoming-software/File2File-Desktop) 桌面端「P2P SSH」连接。

### 下载

| 平台 | 文件 |
|------|------|
| macOS (Apple Silicon) | \`p2pssh-server-darwin-arm64.tar.gz\` |
| Linux amd64 | \`p2pssh-server-linux-amd64.tar.gz\` |
| Linux arm64 | \`p2pssh-server-linux-arm64.tar.gz\` |
| Windows amd64 | \`p2pssh-server-windows-amd64.exe\`（仅 tcp 转发；shell PTY 请用 Linux/macOS） |

### 使用

\`\`\`bash
# Unix
tar -xzf p2pssh-server-<platform>.tar.gz
./p2pssh-server --token YOUR_TOKEN --passwd YOUR_PASSWD --permission YOUR_PERMISSION
\`\`\`

Windows 直接运行 \`p2pssh-server-windows-amd64.exe\`，参数同上。
EOF
}

publish() {
  local repo
  repo="$(resolve_repo)"
  log "仓库: ${repo}"
  log "版本: ${VERSION}  tag: ${TAG}"

  if [[ "${DRY_RUN}" == "1" ]]; then
    log "dry-run：已生成 ${ASSETS_DIR}，跳过 gh release create"
    return 0
  fi

  if gh release view "${TAG}" --repo "${repo}" >/dev/null 2>&1; then
    err "Release ${TAG} 已存在。请改 VERSION，或先删除：gh release delete ${TAG} --repo ${repo} --yes"
    exit 1
  fi

  local args=(
    release create "${TAG}"
    --repo "${repo}"
    --title "${TITLE}"
    --notes "$(release_notes)"
  )
  if [[ "${DRAFT}" == "1" ]]; then
    args+=(--draft)
  fi

  local f
  for f in "${ASSETS_DIR}"/*; do
    [[ -f "${f}" ]] || continue
    args+=("${f}")
  done

  log "正在创建 GitHub Release ${TAG} ..."
  gh "${args[@]}"
  log "发布完成: https://github.com/${repo}/releases/tag/${TAG}"
}

check_prereqs
prepare_assets
publish
