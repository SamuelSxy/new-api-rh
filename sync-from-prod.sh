#!/usr/bin/env bash
# =============================================================================
# sync-from-prod.sh — 拉取生产 PostgreSQL 数据到本地（生产 → 本地）
#
# 用法:
#   ./sync-from-prod.sh                       # 全量同步：drop→restore→restart
#   ./sync-from-prod.sh --tables options      # 只拉指定表(逗号分隔)，不 drop 库
#   ./sync-from-prod.sh --no-backup           # 跳过本地备份(默认会备份)
#   ./sync-from-prod.sh --no-restart          # 同步后不重启本地应用
#   ./sync-from-prod.sh --keep-dump           # 保留 dump 文件(默认清理)
#   ./sync-from-prod.sh --help
#
# 前置：
#   - 本地能 SSH 到生产服务器（私钥登录，默认 root@124.174.124.129）
#   - 本地 docker compose 已起 postgres 容器
#   - 数据通过 SSH stdout→本地 psql stdin 管道传输，dump 不落盘到生产服务器
#
# 安全：dump 含 users.password / tokens.key 等敏感字段，默认放 /tmp 用完即删，
#       切勿提交到 git。如需保留请显式 --keep-dump 并自行放入 .gitignore 目录。
# =============================================================================

set -euo pipefail

# --------------------------- 配置区（按实际改） --------------------------- #
PROD_SERVER="${PROD_SERVER:-root@124.174.124.129}"
PROD_SSH_KEY="${PROD_SSH_KEY:-/home/samuel/.ssh/aigc.pem}"
PROD_PG_CONTAINER="${PROD_PG_CONTAINER:-postgres}"
PROD_PG_USER="${PROD_PG_USER:-root}"
PROD_DB="${PROD_DB:-new-api}"

LOCAL_PG_CONTAINER="${LOCAL_PG_CONTAINER:-postgres}"
LOCAL_PG_USER="${LOCAL_PG_USER:-root}"
LOCAL_DB="${LOCAL_DB:-new-api}"
LOCAL_APP="${LOCAL_APP:-new-api}"   # 同步后重启以清内存缓存
# -------------------------------------------------------------------------- #

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
log()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()   { echo -e "${GREEN}[OK]${NC}    $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC}  $*"; }
err()  { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# --------------------------- 参数解析 --------------------------- #
TABLES=""
NO_BACKUP=false
NO_RESTART=false
KEEP_DUMP=false
for ((i=1; i<=$#; i++)); do
  arg="${!i}"
  case "$arg" in
    --tables)      j=$((i+1)); TABLES="${!j}"; i=$j ;;
    --no-backup)   NO_BACKUP=true ;;
    --no-restart)  NO_RESTART=true ;;
    --keep-dump)   KEEP_DUMP=true ;;
    --help|-h)     sed -n '2,21p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *)             err "未知参数: $arg"; exit 1 ;;
  esac
done

# --------------------------- SSH 包装 --------------------------- #
# 优先使用指定私钥；如未指定但 SSHPASS 存在则降级为密码登录
if [[ -n "${PROD_SSH_KEY:-}" ]]; then
  [[ ! -f "$PROD_SSH_KEY" ]] && { err "SSH 私钥不存在: $PROD_SSH_KEY"; exit 1; }
  chmod 600 "$PROD_SSH_KEY" 2>/dev/null || true
  SSH="ssh -i $PROD_SSH_KEY -o StrictHostKeyChecking=no -o IdentitiesOnly=yes"
elif [[ -n "${SSHPASS:-}" ]]; then
  SSH="sshpass -e ssh -o StrictHostKeyChecking=no"
else
  SSH="ssh -o StrictHostKeyChecking=no"
fi

# --------------------------- 前置检查 --------------------------- #
if ! docker ps --format '{{.Names}}' | grep -qx "$LOCAL_PG_CONTAINER"; then
  err "本地容器未运行: $LOCAL_PG_CONTAINER"; exit 1
fi

# 检查 SSH 连通性
if ! $SSH "$PROD_SERVER" "docker ps --format '{{.Names}}'" >/dev/null 2>&1; then
  err "SSH 到 $PROD_SERVER 失败（检查密钥或 export SSHPASS）"; exit 1
fi

local_psql() { docker exec -i "$LOCAL_PG_CONTAINER" psql -U "$LOCAL_PG_USER" -v ON_ERROR_STOP=1 "$@"; }

# --------------------------- 导出 --------------------------- #
TS=$(date +%Y%m%d_%H%M%S)
DUMP_DIR="/tmp"
DUMP_FILE="$DUMP_DIR/prod_sync_${TS}.sql"

if [[ -n "$TABLES" ]]; then
  # 部分表：用 -t 重复指定，不 drop 库
  TABLE_ARGS=()
  IFS=',' read -ra _tarr <<< "$TABLES"
  for t in "${_tarr[@]}"; do
    [[ -n "$t" ]] && TABLE_ARGS+=( -t "$t" )
  done
  log "从生产拉取表 [$TABLES] → $DUMP_FILE"
  $SSH "$PROD_SERVER" \
    "docker exec $PROD_PG_CONTAINER pg_dump -U $PROD_PG_USER -d $PROD_DB --no-owner --no-acl ${TABLE_ARGS[*]}" \
    > "$DUMP_FILE"
else
  log "从生产全量拉取 → $DUMP_FILE"
  $SSH "$PROD_SERVER" \
    "docker exec $PROD_PG_CONTAINER pg_dump -U $PROD_PG_USER -d $PROD_DB --no-owner --no-acl" \
    > "$DUMP_FILE"
fi

if [[ ! -s "$DUMP_FILE" ]]; then
  err "dump 文件为空，可能 SSH/pg_dump 失败"; exit 1
fi
ok "dump 已生成: $DUMP_FILE ($(du -h "$DUMP_FILE" | cut -f1))"

# --------------------------- 本地备份（可选） --------------------------- #
if [[ "$NO_BACKUP" == false && -z "$TABLES" ]]; then
  BACKUP_FILE="$DUMP_DIR/local_before_restore_${TS}.sql"
  log "备份本地当前数据 → $BACKUP_FILE"
  docker exec "$LOCAL_PG_CONTAINER" pg_dump -U "$LOCAL_PG_USER" -d "$LOCAL_DB" --no-owner --no-acl \
    > "$BACKUP_FILE"
  ok "本地备份完成"
fi

# --------------------------- 恢复 --------------------------- #
if [[ -z "$TABLES" ]]; then
  # 全量恢复：drop & recreate
  log "删库重建 $LOCAL_DB..."
  local_psql -d postgres -c 'DROP DATABASE IF EXISTS "'"$LOCAL_DB"'";'
  local_psql -d postgres -c 'CREATE DATABASE "'"$LOCAL_DB"'";'
  log "导入生产数据..."
  local_psql -d "$LOCAL_DB" < "$DUMP_FILE"
else
  # 部分表：直接导入（会覆盖同名表）
  log "导入部分表..."
  local_psql -d "$LOCAL_DB" < "$DUMP_FILE"
fi
ok "导入完成"

# --------------------------- 重启应用 --------------------------- #
if [[ "$NO_RESTART" == false ]]; then
  if docker ps --format '{{.Names}}' | grep -qx "$LOCAL_APP"; then
    log "重启 $LOCAL_APP 以清内存缓存..."
    docker restart "$LOCAL_APP" >/dev/null && ok "$LOCAL_APP 已重启"
  else
    warn "容器 $LOCAL_APP 未运行，跳过重启"
  fi
else
  warn "已跳过重启，本地可能仍读旧缓存（手动 docker restart $LOCAL_APP）"
fi

# --------------------------- 清理 --------------------------- #
if [[ "$KEEP_DUMP" == true ]]; then
  warn "保留 dump 文件: $DUMP_FILE（含敏感数据，勿提交 git）"
else
  rm -f "$DUMP_FILE"
  [[ "$NO_BACKUP" == false ]] && rm -f "$DUMP_DIR"/local_before_restore_*.sql 2>/dev/null || true
  ok "已清理临时 dump 文件"
fi

ok "同步完成"
