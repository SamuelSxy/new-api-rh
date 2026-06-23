#!/usr/bin/env bash
# =============================================================================
# sync-to-hai.sh — 把默认站(3001)的「模型广场 / 创意工作台」配置增量同步到 hai 站(3002)
#
# 同步语义：按主键 upsert（ON CONFLICT DO UPDATE）
#   - 源(3001)有的行 → 在目标(3002)更新或插入
#   - 目标(3002)独有的行 → 保留，不删除
# 不清空表，可反复运行。
#
# 用法:
#   ./sync-to-hai.sh                      # 同步默认表(models,vendors,studio_*)
#   ./sync-to-hai.sh --tables models      # 只同步指定表(逗号分隔)
#   ./sync-to-hai.sh --with-options       # 额外同步模型广场价格/倍率(options 选定 key)
#   ./sync-to-hai.sh --with-channels      # 额外同步渠道(channels,含API key)+重建abilities
#   ./sync-to-hai.sh --no-restart         # 同步后不重启 hai 应用容器
#   ./sync-to-hai.sh --help
#
# 前置：本脚本需在「同时能访问 postgres 与 postgres-hai 两个容器」的宿主机上运行
#       （即两站同机部署的服务器，或你本机同时跑着两套 compose）。
#       数据通过 docker stdout→stdin 管道传输，两个 PG 不需要在同一 docker 网络。
# =============================================================================

set -euo pipefail

# --------------------------- 配置区（按实际改） --------------------------- #
SRC_PG="postgres"          # 源 PG 容器（3001 默认站）
SRC_USER="root"
SRC_DB="new-api"

DST_PG="postgres-hai"      # 目标 PG 容器（3002 hai 站）
DST_USER="root"
DST_DB="new-api"

DST_APP="new-api-hai"      # 目标应用容器（同步后重启以清内存缓存）

# 默认同步的业务表（主键均为 id）
DEFAULT_TABLES="models vendors studio_model_configs studio_form_schemas"

# --with-options 时同步的 options key（模型广场价格/倍率/分组相关）
OPTION_KEYS="ModelRatio ModelPrice CompletionRatio CacheRatio CreateCacheRatio ImageRatio AudioRatio AudioCompletionRatio VideoInputRatio GroupRatio GroupGroupRatio UserUsableGroups"
# -------------------------------------------------------------------------- #

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
log()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()   { echo -e "${GREEN}[OK]${NC}    $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC}  $*"; }
err()  { echo -e "${RED}[ERROR]${NC} $*" >&2; }

# --------------------------- 参数解析 --------------------------- #
TABLES="$DEFAULT_TABLES"
WITH_OPTIONS=false
WITH_CHANNELS=false
RESTART=true
for ((i=1; i<=$#; i++)); do
  arg="${!i}"
  case "$arg" in
    --tables)        j=$((i+1)); TABLES="${!j//,/ }"; i=$j ;;
    --with-options)  WITH_OPTIONS=true ;;
    --with-channels) WITH_CHANNELS=true ;;
    --no-restart)    RESTART=false ;;
    --help|-h)       sed -n '2,21p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *)               err "未知参数: $arg"; exit 1 ;;
  esac
done

# --------------------------- 前置检查 --------------------------- #
for c in "$SRC_PG" "$DST_PG"; do
  if ! docker ps --format '{{.Names}}' | grep -qx "$c"; then
    err "容器未运行: $c"; exit 1
  fi
done

dst_psql() { docker exec -i "$DST_PG" psql -U "$DST_USER" -d "$DST_DB" -v ON_ERROR_STOP=1 "$@"; }
src_meta() { docker exec "$SRC_PG" psql -U "$SRC_USER" -d "$SRC_DB" -tAc "$1"; }
dst_meta() { docker exec "$DST_PG" psql -U "$DST_USER" -d "$DST_DB" -tAc "$1"; }

# 同步一张表
#   $1=表名 $2=冲突主键列名(upsert用) $3=源端 WHERE 过滤(可空) $4=模式 upsert|replace
#   upsert : 按主键 ON CONFLICT DO UPDATE，保留目标独有行
#   replace: TRUNCATE 后整表灌入（用于 abilities 这类派生表）
sync_table() {
  local t="$1" pk="${2:-}" where="${3:-}" mode="${4:-upsert}"

  # 校验表存在
  local exists
  exists=$(dst_meta "SELECT to_regclass('public.$t') IS NOT NULL;")
  if [[ "$exists" != "t" ]]; then warn "目标库无表 $t，跳过"; return; fi

  # 取目标表列（按物理顺序，quote_ident 自动给保留字加引号）
  local cols
  cols=$(dst_meta "SELECT string_agg(quote_ident(column_name), ',' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema='public' AND table_name='$t';")
  if [[ -z "$cols" ]]; then warn "$t 无列信息，跳过"; return; fi

  local src_select="SELECT $cols FROM public.$t"
  [[ -n "$where" ]] && src_select="$src_select WHERE $where"

  log "同步 $t ($mode)..."
  if [[ "$mode" == "replace" ]]; then
    {
      printf 'BEGIN;\n'
      printf 'TRUNCATE TABLE public.%s RESTART IDENTITY;\n' "$t"
      printf '\\copy public.%s (%s) FROM STDIN WITH CSV\n' "$t" "$cols"
      docker exec "$SRC_PG" psql -U "$SRC_USER" -d "$SRC_DB" -v ON_ERROR_STOP=1 \
        -c "COPY ($src_select) TO STDOUT WITH CSV"
      printf '\\.\n'
      printf 'COMMIT;\n'
    } | dst_psql >/dev/null
  else
    local upd
    upd=$(dst_meta "SELECT string_agg(quote_ident(column_name)||'=EXCLUDED.'||quote_ident(column_name), ',' ORDER BY ordinal_position) FROM information_schema.columns WHERE table_schema='public' AND table_name='$t' AND column_name <> '$pk';")
    {
      printf 'BEGIN;\n'
      printf 'CREATE TEMP TABLE _sync (LIKE public.%s INCLUDING DEFAULTS) ON COMMIT DROP;\n' "$t"
      printf '\\copy _sync (%s) FROM STDIN WITH CSV\n' "$cols"
      docker exec "$SRC_PG" psql -U "$SRC_USER" -d "$SRC_DB" -v ON_ERROR_STOP=1 \
        -c "COPY ($src_select) TO STDOUT WITH CSV"
      printf '\\.\n'
      printf 'INSERT INTO public.%s (%s) SELECT %s FROM _sync ON CONFLICT (%s) DO UPDATE SET %s;\n' \
        "$t" "$cols" "$cols" "$pk" "$upd"
      # 仅对整型 id 主键修正自增序列
      if [[ "$pk" == "id" ]]; then
        printf "SELECT setval(s.seq, GREATEST(m.mx,1)) FROM (SELECT pg_get_serial_sequence('public.%s','id') AS seq) s, (SELECT COALESCE(MAX(id),0) AS mx FROM public.%s) m WHERE s.seq IS NOT NULL;\n" "$t" "$t"
      fi
      printf 'COMMIT;\n'
    } | dst_psql >/dev/null
  fi
  ok "$t 同步完成"
}

# --------------------------- 执行 --------------------------- #
log "源: ${SRC_PG}/${SRC_DB}  →  目标: ${DST_PG}/${DST_DB}"
log "表: ${TABLES}$( [[ "$WITH_OPTIONS" == true ]] && echo " + options(选定key)" )"

for t in $TABLES; do
  sync_table "$t" "id"
done

if [[ "$WITH_CHANNELS" == true ]]; then
  warn "⚠ --with-channels 会把 3001 的渠道(含上游 API key/密钥/base_url)同步到 3002！"
  # channels: 按 id upsert（保留 3002 独有渠道）
  sync_table "channels" "id" "" "upsert"
  # abilities: 派生表，整表替换（与 FixAbility 同理，TRUNCATE 后重灌）
  sync_table "abilities" "" "" "replace"
fi

if [[ "$WITH_OPTIONS" == true ]]; then
  # 构造 key IN ('a','b',...)
  keys_in=$(printf "'%s'," $OPTION_KEYS); keys_in="${keys_in%,}"
  sync_table "options" "key" "\"key\" IN ($keys_in)"
fi

if [[ "$RESTART" == true ]]; then
  log "重启 $DST_APP 以清内存缓存..."
  docker restart "$DST_APP" >/dev/null && ok "$DST_APP 已重启"
else
  warn "已跳过重启，目标站可能仍读旧缓存（手动 docker restart $DST_APP）"
fi

ok "全部完成"
