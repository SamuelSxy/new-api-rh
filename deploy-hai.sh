#!/usr/bin/env bash
# =============================================================================
# deploy-hai.sh — 构建并发布 hai 主题站点到生产服务器（与默认站点同机、独立目录）
# 用法: ./deploy-hai.sh [选项]
#
# 选项:
#   --no-cache    强制完整重建（不使用 Docker 缓存）
#   --skip-build  跳过构建，直接使用现有镜像
#   --help        显示帮助
#
# 前置: 服务器 ${SERVER_DIR} 下需有本仓库的 docker-compose.hai.yml（命名为 docker-compose.yml）。
# =============================================================================

set -euo pipefail

# =============================================================================
# ★ 配置区域 — 按实际情况修改
# =============================================================================
SERVER_USER="root"
SERVER_HOST="124.174.124.129"
SERVER_DIR="/opt/new-api-hai"                      # hai 站点独立目录（不要与默认站点 /opt/new-api-rh 相同）
SSH_KEY="/home/samuel/.ssh/aigc.pem"  # 留空使用默认 ~/.ssh/id_rsa，或填绝对路径如 ~/.ssh/my_key
SERVER_PASSWORD="${SERVER_PASSWORD:-}"            # 不要在文件中硬编码！通过环境变量提供：export SERVER_PASSWORD=... （需 sshpass）

LOCAL_IMAGE="new-api-hai"                          # 与 docker-compose.hai.yml 中 image 一致
PROD_TAG="prod"
TMP_FILE="/tmp/new-api-hai-deploy.tar.gz"
ACCESS_URL="http://hai.ai-gc.net/"                # 部署完成提示用，按 hai 实际域名修改
# =============================================================================

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
log_success() { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# 解析参数
NO_CACHE=false
SKIP_BUILD=false
for arg in "$@"; do
  case "$arg" in
    --no-cache)   NO_CACHE=true ;;
    --skip-build) SKIP_BUILD=true ;;
    --help)
      sed -n '2,13p' "$0" | sed 's/^# //'
      exit 0 ;;
    *) log_error "未知参数: $arg"; exit 1 ;;
  esac
done

# 校验配置
if [[ "$SERVER_HOST" == "your-server-ip" ]]; then
  log_error "请先编辑 deploy-hai.sh，填写 SERVER_HOST 等配置项"
  exit 1
fi

# SSH 选项
SSH_OPTS=(-o StrictHostKeyChecking=no -o ConnectTimeout=10 -o ServerAliveInterval=30 -o ServerAliveCountMax=20)
[[ -n "$SSH_KEY" ]] && SSH_OPTS+=(-i "$SSH_KEY")
SCP_OPTS=("${SSH_OPTS[@]}")

# 账密登录：使用 sshpass 包装 ssh/scp
SSH_CMD="ssh"
SCP_CMD="scp"
if [[ -n "$SERVER_PASSWORD" ]]; then
  if ! command -v sshpass &>/dev/null; then
    log_error "账密登录需要安装 sshpass"
    log_error "  macOS: brew install sshpass"
    log_error "  Ubuntu/Debian: apt install sshpass"
    exit 1
  fi
  SSH_CMD="sshpass -p ${SERVER_PASSWORD} ssh"
  SCP_CMD="sshpass -p ${SERVER_PASSWORD} scp"
fi

SERVER="${SERVER_USER}@${SERVER_HOST}"

# --------------------------------------------------------------------------- #
# Step 1: 构建 hai 镜像（THEME=hai 通过 --build-arg 注入）
# --------------------------------------------------------------------------- #
if [[ "$SKIP_BUILD" == false ]]; then
  # 自动写入版本号：优先用 git tag，否则用 commit 短 hash
  VERSION=$(git describe --tags --always 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo "unknown")
  echo "$VERSION" > VERSION
  log_info "Step 1/5  构建 hai Docker 镜像（版本: ${VERSION}）..."
  BUILD_ARGS=()
  [[ "$NO_CACHE" == true ]] && BUILD_ARGS+=(--no-cache) && log_warn "使用 --no-cache，将完整重建（约 5~10 分钟）"
  docker build "${BUILD_ARGS[@]}" --network host --build-arg THEME=hai -t "${LOCAL_IMAGE}:latest" .
  log_success "镜像构建完成"
else
  log_warn "Step 1/5  已跳过构建，使用现有镜像"
fi

# --------------------------------------------------------------------------- #
# Step 2: 打 prod 标签
# --------------------------------------------------------------------------- #
log_info "Step 2/5  打 prod 标签..."
docker tag "${LOCAL_IMAGE}:latest" "${LOCAL_IMAGE}:${PROD_TAG}"
log_success "标签: ${LOCAL_IMAGE}:${PROD_TAG}"

# --------------------------------------------------------------------------- #
# Step 3+4: 打包并流式上传到服务器（pv 显示进度）
# --------------------------------------------------------------------------- #
log_info "Step 3/5  打包镜像 → ${TMP_FILE}"
docker save "${LOCAL_IMAGE}:latest" "${LOCAL_IMAGE}:${PROD_TAG}" | gzip > "${TMP_FILE}"
SIZE=$(du -sh "${TMP_FILE}" | cut -f1)
SIZE_BYTES=$(stat -c%s "${TMP_FILE}" 2>/dev/null || stat -f%z "${TMP_FILE}" 2>/dev/null || echo 0)
log_success "打包完成，大小: ${SIZE}"

log_info "Step 4/5  上传到 ${SERVER}（${SIZE}）..."
if command -v pv &>/dev/null; then
  pv -s "${SIZE_BYTES}" "${TMP_FILE}" | $SSH_CMD "${SSH_OPTS[@]}" "${SERVER}" 'cat > ~/new-api-hai-deploy.tar.gz'
else
  log_warn "未安装 pv，无进度显示（apt install pv 可启用）"
  $SCP_CMD "${SCP_OPTS[@]}" "${TMP_FILE}" "${SERVER}:~/new-api-hai-deploy.tar.gz"
fi
rm -f "${TMP_FILE}"
log_success "上传完成"

# --------------------------------------------------------------------------- #
# Step 5: 服务器端加载镜像并重启
# --------------------------------------------------------------------------- #
log_info "Step 5/5  服务器加载镜像并重启..."
$SSH_CMD "${SSH_OPTS[@]}" "${SERVER}" bash -s << EOF
  set -e
  echo "[server] 加载镜像..."
  docker load < ~/new-api-hai-deploy.tar.gz
  docker tag "${LOCAL_IMAGE}:${PROD_TAG}" "${LOCAL_IMAGE}:latest"
  # rm -f ~/new-api-hai-deploy.tar.gz

  echo "[server] 重启容器..."
  cd "${SERVER_DIR}"
  docker compose up -d --force-recreate --pull never

  echo "[server] 清理悬空镜像..."
  docker image prune -f

  echo "[server] 当前运行容器:"
  docker ps --format "table {{.Names}}\t{{.Image}}\t{{.Status}}"
EOF

log_success "====================================="
log_success " hai 站点部署完成！"
log_success " 访问: ${ACCESS_URL}"
log_success "====================================="
