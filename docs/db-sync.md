# 数据库同步：本地 → 生产

本项目使用 PostgreSQL，本地和生产均通过 Docker 容器运行。

- 本地容器名：`postgres`（`docker-compose.yml`）
- 生产服务器：`root@114.215.172.94`，目录：`/var/www/html/new-api-rh`

> **安全提示**：所有 SSH 凭据必须通过环境变量或 SSH 密钥提供，**严禁**在脚本、命令行或文档中明文硬编码密码。推荐使用 `ssh-copy-id` 配置密钥登录。如必须使用密码，请在当前 shell 临时导出 `SSHPASS`（不要写入 `.bashrc`、不要提交到 git）：
>
> ```bash
> read -rs -p "Production SSH password: " SSHPASS && export SSHPASS
> ```

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

推荐使用 SSH 密钥：

```bash
scp ./local_backup_*.sql root@114.215.172.94:/tmp/db_import.sql
```

若必须使用密码，从环境变量读取（`-e` 表示从 `$SSHPASS` 读取）：

```bash
sshpass -e scp -o StrictHostKeyChecking=no \
  ./local_backup_*.sql \
  root@114.215.172.94:/tmp/db_import.sql
```

---

## 第三步：在生产服务器上恢复

SSH 登录生产服务器：

```bash
ssh root@114.215.172.94
# 或使用环境变量中的密码：
# sshpass -e ssh -o StrictHostKeyChecking=no root@114.215.172.94
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

脚本从环境变量读取凭据，不在源码中硬编码。运行前请先 `export SSHPASS=...`，或配置 SSH 密钥后删除 `sshpass` 调用。

```bash
#!/usr/bin/env bash
set -euo pipefail

SERVER="${PROD_SERVER:-root@114.215.172.94}"
BACKUP="./db_sync_$(date +%Y%m%d_%H%M%S).sql"

# 凭据：优先使用 SSH 密钥；若需密码登录，请通过环境变量 SSHPASS 提供
if [[ -n "${SSHPASS:-}" ]]; then
  export SSHPASS
  SSH="sshpass -e ssh -o StrictHostKeyChecking=no"
  SCP="sshpass -e scp -o StrictHostKeyChecking=no"
else
  SSH="ssh"
  SCP="scp"
fi

cleanup() { unset SSHPASS; rm -f "$BACKUP"; }
trap cleanup EXIT

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
```

---

## 注意事项

| 情况 | 说明 |
|------|------|
| 数据库名含 `-` | psql 中必须用单引号包裹双引号：`-c 'DROP DATABASE "new-api";'` |
| 只同步特定表 | `pg_dump` 加 `-t table_name` 参数 |
| 增量同步 | 不支持，每次均为全量覆盖 |
| 同步前确认生产无活跃连接 | 可用 `docker compose stop new-api` 先停应用再操作 |
| 凭据管理 | 严禁硬编码；使用 SSH 密钥或 `SSHPASS` 环境变量；切勿提交到 git |
