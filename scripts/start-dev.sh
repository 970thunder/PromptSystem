#!/usr/bin/env bash
# ============================================================================
# PromptOS 一键启动脚本（Windows Git Bash / Linux/macOS 通用）
#
# 固定端口段：28301 - 28399（避免与本机其他应用抢端口）
#   28301  前端 Vite dev server
#   28302  后端 Go API
#   28303  MySQL（容器 3306 -> 主机 28303）
#   28304  Redis（容器 6379  -> 主机 28304）
#
# 用法：
#   bash scripts/start-dev.sh         启动全部服务
#   bash scripts/start-dev.sh --no-db 跳过 MySQL/Redis（使用后端内存降级）
#   bash scripts/start-dev.sh stop    停止全部服务
# ============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="$ROOT_DIR/logs"
PID_DIR="$LOG_DIR/.pids"
UPLOAD_DIR="$ROOT_DIR/data/uploads"

# ---- 固定端口分配（28301-28399 段内） ----
FRONTEND_PORT=28301
BACKEND_PORT=28302
MYSQL_PORT=28303
REDIS_PORT=28304

# ---- 日志/进程文件 ----
FRONTEND_LOG="$LOG_DIR/frontend.log"
BACKEND_LOG="$LOG_DIR/backend.log"
FRONTEND_PID="$PID_DIR/frontend.pid"
BACKEND_PID="$PID_DIR/backend.pid"

mkdir -p "$LOG_DIR" "$PID_DIR" "$UPLOAD_DIR" 2>/dev/null || true

# ============================================================================
# 工具函数
# ============================================================================
log_info()  { printf '[INFO ] %s\n' "$*"; }
log_ok()    { printf '[ OK  ] %s\n' "$*"; }
log_warn()  { printf '[WARN ] %s\n' "$*" >&2; }
log_error() { printf '[ERROR] %s\n' "$*" >&2; }

is_windows() { [[ "$OSTYPE" == "msys"* || "$OSTYPE" == "cygwin"* || -n "${WINDIR:-}" ]]; }

# 查找监听指定端口的进程 PID 列表（Windows: netstat / macOS: lsof）
port_pids() {
  local port="$1"
  if is_windows; then
    netstat -ano | awk -v p=":$port" '$0 ~ p && $0 ~ /LISTENING/ { print $NF }' | sort -u
  else
    lsof -ti tcp:"$port" 2>/dev/null || true
  fi
}

# 检查端口是否被占用；占用则报错退出（绝不静默换端口）
check_port() {
  local port="$1" name="$2"
  local pids
  pids=$(port_pids "$port" || true)
  if [[ -n "${pids//[[:space:]]/}" ]]; then
    log_error "端口 $port（$name）已被占用，请先释放后重试："
    for pid in $pids; do
      log_error "  PID $pid"
    done
    exit 1
  fi
  log_ok "端口 $port（$name）可用"
}

is_running() { [[ -f "$1" ]] && kill -0 "$(cat "$1" 2>/dev/null)" 2>/dev/null; }

kill_pid_file() {
  local pid_file="$1" name="$2"
  if [[ ! -f "$pid_file" ]]; then
    return 0
  fi
  local pid
  pid=$(cat "$pid_file" 2>/dev/null || true)
  if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
    log_info "停止 $name (PID $pid)"
    kill "$pid" 2>/dev/null || true
    sleep 1
    kill -9 "$pid" 2>/dev/null || true
  fi
  # 沙箱可能拦截 rm（转回收站），失败不能中断脚本（set -e 下需 || true）
  rm -f "$pid_file" 2>/dev/null || true
}

# Windows 上 npm/go run 的子进程可能与记录 PID 分离，stop 时按端口兜底清理
# 残留的监听进程（只清理 28301-28304 固定段，绝不触碰其他端口）。
kill_port_listeners() {
  local port="$1" name="$2"
  local pids
  pids=$(port_pids "$port" || true)
  if [[ -n "${pids//[[:space:]]/}" ]]; then
    for pid in $pids; do
      log_info "清理 $name 残留进程 (PID $pid, 端口 $port)"
      if is_windows; then
        # Git Bash 下需要禁止 MSYS 路径转换（否则 /F 会被当成盘符路径），
        # 单斜杠参数 + MSYS_NO_PATHCONV=1 是 Windows taskkill 的可靠写法。
        MSYS_NO_PATHCONV=1 taskkill /F /T /PID "$pid" >/dev/null 2>&1 || true
      else
        kill -9 "$pid" 2>/dev/null || true
      fi
    done
  fi
}

wait_ready() {
  local url="$1" name="$2" max="$3"
  for _ in $(seq 1 "$max"); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

# ============================================================================
# stop：停止全部服务
# ============================================================================
stop_all() {
  log_info "停止 PromptOS 全部服务..."
  kill_pid_file "$FRONTEND_PID" "前端 (28301)"
  kill_pid_file "$BACKEND_PID"  "后端 (28302)"
  kill_port_listeners "$FRONTEND_PORT" "前端 (28301)"
  kill_port_listeners "$BACKEND_PORT"  "后端 (28302)"

  if docker compose version >/dev/null 2>&1; then
    (cd "$ROOT_DIR" && docker compose down 2>/dev/null || true)
  fi
  log_ok "已停止（MySQL/Redis 容器已 down；数据卷保留）"
  exit 0
}

# ============================================================================
# main
# ============================================================================
if [[ "${1:-}" == "stop" ]]; then
  stop_all
fi

if [[ "${1:-}" == "--no-db" ]]; then
  SKIP_DB=1
else
  SKIP_DB=0
fi

log_info "PromptOS 一键启动（固定端口段 28301-28399）"
log_info "  前端 http://localhost:$FRONTEND_PORT"
log_info "  后端 http://localhost:$BACKEND_PORT"

# 0) 若已有服务在跑，先提示
if is_running "$FRONTEND_PID" || is_running "$BACKEND_PID"; then
  log_error "检测到 PromptOS 服务已在运行，请先执行：bash scripts/start-dev.sh stop"
  exit 1
fi

# 1) 检查端口占用
check_port "$FRONTEND_PORT" "前端 Vite"
check_port "$BACKEND_PORT"  "后端 Go API"
if [[ "$SKIP_DB" == "0" ]]; then
  check_port "$MYSQL_PORT" "MySQL"
  check_port "$REDIS_PORT" "Redis"
fi

# 2) MySQL + Redis（容器）
if [[ "$SKIP_DB" == "0" ]]; then
  if ! docker compose version >/dev/null 2>&1; then
    log_error "未检测到 Docker Compose；请安装 Docker 或使用 --no-db 模式（内存降级）"
    exit 1
  fi
  log_info "启动 MySQL (28303) + Redis (28304) 容器..."
  (
    cd "$ROOT_DIR"
    PROMPTOS_MYSQL_PORT=$MYSQL_PORT \
    PROMPTOS_REDIS_PORT=$REDIS_PORT \
      docker compose up -d mysql redis
  )
fi

# 3) 后端 Go API
log_info "启动后端 API (28302)..."
(
  cd "$ROOT_DIR/src/backend"
  APP_ENV=development \
  PORT=$BACKEND_PORT \
  MYSQL_HOST=127.0.0.1 \
  MYSQL_PORT=$MYSQL_PORT \
  MYSQL_USER=root \
  MYSQL_PASSWORD=root \
  MYSQL_DATABASE=promptos \
  REDIS_HOST=127.0.0.1 \
  REDIS_PORT=$REDIS_PORT \
  UPLOAD_DIR="$UPLOAD_DIR" \
  UPLOAD_BASE_URL="http://localhost:$BACKEND_PORT" \
  JWT_SECRET="${PROMPTOS_JWT_SECRET:-promptos-local-dev-secret-change-me-28302}" \
  ALLOWED_ORIGIN="http://localhost:$FRONTEND_PORT" \
    nohup go run ./cmd/api > "$BACKEND_LOG" 2>&1 &
  echo $! > "$BACKEND_PID"
)

# 4) 前端 Vite
log_info "启动前端 dev server (28301)..."
(
  cd "$ROOT_DIR/src/frontend"
  PROMPTOS_FRONTEND_PORT=$FRONTEND_PORT \
  PROMPTOS_BACKEND_PORT=$BACKEND_PORT \
    nohup npm run dev > "$FRONTEND_LOG" 2>&1 &
  echo $! > "$FRONTEND_PID"
)

# 5) 等待就绪
log_info "等待服务就绪..."
if wait_ready "http://localhost:$BACKEND_PORT/api/v1/health/live" "后端" 60; then
  log_ok "后端 API 就绪：http://localhost:$BACKEND_PORT"
else
  log_error "后端启动超时，请查看 $BACKEND_LOG"
  exit 1
fi

if wait_ready "http://localhost:$FRONTEND_PORT/" "前端" 60; then
  log_ok "前端就绪：http://localhost:$FRONTEND_PORT"
else
  log_error "前端启动超时，请查看 $FRONTEND_LOG"
  exit 1
fi

# 6) 摘要
cat <<EOF

============================================================
  PromptOS 已启动
  - 前端    http://localhost:$FRONTEND_PORT
  - 后端    http://localhost:$BACKEND_PORT/api/v1
  - 健康检查 http://localhost:$BACKEND_PORT/api/v1/health/ready
  - MySQL   localhost:$MYSQL_PORT
  - Redis   localhost:$REDIS_PORT

  日志：$LOG_DIR/（frontend.log / backend.log）
  停止：bash scripts/start-dev.sh stop
============================================================
EOF
