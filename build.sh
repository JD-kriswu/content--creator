#!/bin/bash
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== [1/2] 构建前端 ==="
cd "$ROOT/frontend"
npm install
npm run build

echo "=== [2/2] 构建后端 ==="
cd "$ROOT/backend"
go build -o "$ROOT/content-creator-imm" .

echo "=== ✅ 构建完成 ==="
echo "  前端: frontend/dist/"
echo "  后端: content-creator-imm"
