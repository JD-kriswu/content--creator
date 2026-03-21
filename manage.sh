#!/bin/bash
set -e
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="$ROOT/content-creator-imm"
PID_FILE="$ROOT/server.pid"
LOG_FILE="$ROOT/server.log"
CONFIG="$ROOT/backend/config.json"

usage() {
  echo "用法: $0 <command>"
  echo ""
  echo "Commands:"
  echo "  start                              启动后端服务"
  echo "  stop                               停止后端服务"
  echo "  restart                            重启后端服务"
  echo "  status                             查看服务状态"
  echo "  logs                               查看日志 (tail -f)"
  echo "  add-user <username> <email> <pwd>  添加用户"
  echo "  list-users                         列出所有用户"
  exit 1
}

cmd_start() {
  if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "服务已在运行 (PID $(cat "$PID_FILE"))"
    exit 0
  fi
  if [ ! -f "$BINARY" ]; then
    echo "❌ 未找到二进制文件，请先运行 ./build.sh"
    exit 1
  fi
  cd "$ROOT/backend"
  nohup "$BINARY" >> "$LOG_FILE" 2>&1 &
  echo $! > "$PID_FILE"
  sleep 1
  if kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "✅ 服务已启动 (PID $(cat "$PID_FILE"))"
  else
    echo "❌ 启动失败，查看日志: $LOG_FILE"
    exit 1
  fi
}

cmd_stop() {
  if [ ! -f "$PID_FILE" ]; then
    echo "服务未运行"
    return
  fi
  PID=$(cat "$PID_FILE")
  if kill -0 "$PID" 2>/dev/null; then
    kill "$PID"
    rm -f "$PID_FILE"
    echo "✅ 服务已停止"
  else
    rm -f "$PID_FILE"
    echo "服务未运行（已清理旧 PID 文件）"
  fi
}

cmd_status() {
  if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
    echo "✅ 运行中 (PID $(cat "$PID_FILE"))"
  else
    echo "⏹  未运行"
  fi
}

cmd_add_user() {
  [ $# -lt 3 ] && { echo "用法: $0 add-user <username> <email> <password>"; exit 1; }
  PORT=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('port','3004'))" 2>/dev/null || echo "3004")
  curl -s -X POST "http://localhost:$PORT/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$1\",\"email\":\"$2\",\"password\":\"$3\"}" | python3 -m json.tool
}

cmd_list_users() {
  DB_HOST=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('db_host','127.0.0.1'))" 2>/dev/null || echo "127.0.0.1")
  DB_PORT=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('db_port','3306'))" 2>/dev/null || echo "3306")
  DB_USER=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('db_user','root'))" 2>/dev/null || echo "root")
  DB_PASS=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('db_password',''))" 2>/dev/null || echo "")
  DB_NAME=$(python3 -c "import json; d=json.load(open('$CONFIG')); print(d.get('db_name','content_creator'))" 2>/dev/null || echo "content_creator")
  mysql -h"$DB_HOST" -P"$DB_PORT" -u"$DB_USER" -p"$DB_PASS" "$DB_NAME" \
    -e "SELECT id, username, email, role, active, created_at FROM users ORDER BY id;"
}

case "${1:-}" in
  start)      cmd_start ;;
  stop)       cmd_stop ;;
  restart)    cmd_stop; sleep 1; cmd_start ;;
  status)     cmd_status ;;
  logs)       tail -f "$LOG_FILE" ;;
  add-user)   shift; cmd_add_user "$@" ;;
  list-users) cmd_list_users ;;
  *)          usage ;;
esac
