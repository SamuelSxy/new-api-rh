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

## 反向同步：生产 → 本地

排查线上问题（如计费异常、配置回滚）时常用。流程与上面相反，但更轻量——本地数据通常可以随时丢弃，不需要那么谨慎。

### 1. 从生产导出

```bash
# 在本地机器执行，通过 SSH 把远端 pg_dump 输出直接拉到本地文件
ssh root@114.215.172.94 \
  "docker exec postgres pg_dump -U root -d new-api --no-owner --no-acl" \
  > ./prod_backup_$(date +%Y%m%d_%H%M%S).sql
```

要点：
- `pg_dump` 在生产容器内执行，输出经 SSH 直接重定向到本地文件，无需先落盘到服务器。
- `--no-owner --no-acl` 跳过 owner/权限声明，避免本地恢复时因角色不存在报错。

### 2. 本地恢复

```bash
# 可选：备份本地当前数据
docker exec postgres pg_dump -U root -d new-api --no-owner --no-acl \
  > ./local_before_restore_$(date +%Y%m%d_%H%M%S).sql

# 删库重建
docker exec -i postgres psql -U root -d postgres \
  -c 'DROP DATABASE IF EXISTS "new-api";'
docker exec -i postgres psql -U root -d postgres \
  -c 'CREATE DATABASE "new-api";'

# 导入生产数据
docker exec -i postgres psql -U root -d new-api \
  < ./prod_backup_*.sql

# 重启本地应用
docker compose restart new-api
```

### 3. 只拉部分表（轻量排查）

排查特定问题（如定价配置、渠道表）时无需全量，`pg_dump` 加 `-t` 指定表即可：

```bash
ssh root@114.215.172.94 \
  "docker exec postgres pg_dump -U root -d new-api --no-owner --no-acl \
   -t options -t channels -t tokens -t users -t abilities" \
  > ./prod_partial_$(date +%Y%m%d_%H%M%S).sql
```

恢复时**不要 DROP DATABASE**，直接导入即可（会覆盖同名表）：

```bash
docker exec -i postgres psql -U root -d new-api \
  < ./prod_partial_*.sql
```

### 4. 安全提示

- 生产 dump 里的 `users.password`、`tokens.key` 等敏感字段会一并落到本地，**不要提交到 git**，建议放 `/tmp` 用完即删，或加入 `.gitignore`。
- 本地若是 SQLite/MySQL（看 `docker-compose.yml` 的 DB 配置），不能直接恢复 PostgreSQL dump，需先转格式或临时起 PG 容器。

---

## 注意事项

| 情况 | 说明 |
|------|------|
| 数据库名含 `-` | psql 中必须用单引号包裹双引号：`-c 'DROP DATABASE "new-api";'` |
| 只同步特定表 | `pg_dump` 加 `-t table_name` 参数 |
| 增量同步 | 不支持，每次均为全量覆盖 |
| 同步前确认生产无活跃连接 | 可用 `docker compose stop new-api` 先停应用再操作 |
| 凭据管理 | 严禁硬编码；使用 SSH 密钥或 `SSHPASS` 环境变量；切勿提交到 git |
