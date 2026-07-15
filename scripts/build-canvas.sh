#!/usr/bin/env bash
# 构建 infinite-canvas 前端 SPA 产物，供 Go 通过 embed.FS 内嵌。
# 需要先 clone infinite-canvas 仓库到仓库根的 infinite-canvas/ 目录。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
CANVAS_DIR="$ROOT_DIR/infinite-canvas/web"

if [ ! -d "$CANVAS_DIR" ]; then
    echo "错误：未找到 $CANVAS_DIR" >&2
    echo "请先 clone infinite-canvas 仓库：" >&2
    echo "  git clone git@github.com:SamuelSxy/infinite-canvas.git infinite-canvas" >&2
    exit 1
fi

cd "$CANVAS_DIR"

# 用 VITE_BASE_PATH 让 vite 把资源路径前缀换成 /canvas/，
# 与 new-api 后端 /canvas 路由对齐。
export VITE_BASE_PATH="/canvas/"

echo "==> 安装依赖 (bun install)"
bun install --frozen-lockfile

echo "==> 构建产物 (bun run build)"
bun run build

echo "==> 完成：$CANVAS_DIR/dist"
