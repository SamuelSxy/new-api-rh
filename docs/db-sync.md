# 数据库同步：本地 → 生产

本项目使用 PostgreSQL，本地和生产均通过 Docker 容器运行。

- 本地容器名：`postgres`（`docker-compose.yml`）
- 生产服务器：`root@114.215.172.94`，目录：`/var/www/html/new-api-rh`

---

## 第一步：导出本地数据库

在本地项目目录执行：

```bash
docker exec postgres pg_dump -U root -d new-api \
  --no-owner --no-acl \
  > ./local_backup_$(date +%Y%m%d_%H%M%S).sql
```

生成文件如 `local_backup_20260529_122058.sql`。

---

## 第二步：上传到生产服务器

```bash
sshpass -p '***REMOVED***' scp \
  -o StrictHostKeyChecking=no \
  ./local_backup_*.sql \
  root@114.215.172.94:/tmp/db_import.sql
```

---

## 第三步：在生产服务器上恢复

SSH 登录生产服务器：

```bash
sshpass -p '***REMOVED***' ssh -o StrictHostKeyChecking=no root@114.215.172.94
```

执行以下命令：

```bash
# 1. 备份生产现有数据（可选但推荐）
docker exec postgres pg_dump -U root -d new-api \
  --no-owner --no-acl > /tmp/prod_backup_before_sync.sql

# 2. 删除并重建数据库（注意：数据库名含连字符需加双引号）
docker exec -i postgres psql -U root -d postgres \
  -c 'DROP DATABASE IF EXISTS "new-api";'
docker exec -i postgres psql -U root -d postgres \
  -c 'CREATE DATABASE "new-api";'

# 3. 导入本地数据
docker exec -i postgres psql -U root -d new-api \
  < /tmp/db_import.sql

# 4. 重启应用
cd /var/www/html/new-api-rh
docker compose restart new-api
```

---

## 一键脚本（本地执行）

```bash
#!/usr/bin/env bash
set -euo pipefail

SERVER="root@114.215.172.94"
PASS="***REMOVED***"
SSH="sshpass -p ${PASS} ssh -o StrictHostKeyChecking=no"
SCP="sshpass -p ${PASS} scp -o StrictHostKeyChecking=no"
BACKUP="./db_sync_$(date +%Y%m%d_%H%M%S).sql"

echo "[1/4] 导出本地数据库..."
docker exec postgres pg_dump -U root -d new-api --no-owner --no-acl > "$BACKUP"

echo "[2/4] 上传到生产服务器..."
$SCP "$BACKUP" "${SERVER}:/tmp/db_import.sql"

echo "[3/4] 生产服务器恢复数据..."
$SSH "$SERVER" bash -s << 'EOF'
  set -e
  docker exec -i postgres psql -U root -d postgres -c 'DROP DATABASE IF EXISTS "new-api";'
  docker exec -i postgres psql -U root -d postgres -c 'CREATE DATABASE "new-api";'
  docker exec -i postgres psql -U root -d new-api < /tmp/db_import.sql
  rm -f /tmp/db_import.sql
EOF

echo "[4/4] 重启生产应用..."
$SSH "$SERVER" "cd /var/www/html/new-api-rh && docker compose restart new-api"

echo "同步完成！"
rm -f "$BACKUP"
```

---

## 注意事项

| 情况 | 说明 |
|------|------|
| 数据库名含 `-` | psql 中必须用单引号包裹双引号：`-c 'DROP DATABASE "new-api";'` |
| 只同步特定表 | `pg_dump` 加 `-t table_name` 参数 |
| 增量同步 | 不支持，每次均为全量覆盖 |
| 同步前确认生产无活跃连接 | 可用 `docker compose stop new-api` 先停应用再操作 |
