#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

cmd="${1:-}" || true
shift || true

# 可选输出控制：默认静默，使用 -v/--verbose 打开详细输出，--log-file 指定日志文件
VERBOSE=0
LOG_FILE="${LOG_FILE:-logs/e2e_test.log}"
# 解析附加参数（例如：./scripts/e2e_test.sh run -v --log-file logs/custom.log）
while ((${#:-0})); do
  case "${1:-}" in
    -v|--verbose)
      VERBOSE=1; shift || true ;;
    -q|--quiet)
      VERBOSE=0; shift || true ;;
    --log-file)
      LOG_FILE="${2:-$LOG_FILE}"; shift 2 || true ;;
    *)
      # 其他参数忽略
      shift || true ;;
  esac
  (( ${#:-0} > 0 )) || break
done

# 确保日志目录存在
mkdir -p "$(dirname "$LOG_FILE")" || true

start_containers() {
  echo "Starting Redis and MySQL containers (if not running)..."
  docker start redis 2>/dev/null || docker run -d --name redis -p 6379:6379 redis:alpine
  docker start mysql 2>/dev/null || docker run -d --name mysql -e MYSQL_ROOT_PASSWORD=azsx0123456 -p 3307:3306 mysql:8.0
}

stop_containers() {
  echo "Stopping Redis and MySQL containers..."
  docker stop redis mysql >/dev/null 2>&1 || true
}

# 新增：一个更强大的辅助函数，用于杀死占用指定端口的进程
kill_process_on_port() {
  local port=$1
  local service_name=$2
  echo "Checking for any process using port ${port} (${service_name})..."
  # 使用 lsof 查找 PID。-t 选项只输出 PID。
  # || true 是为了在没有进程找到时防止脚本因 set -e 而退出
  local pid
  pid=$(lsof -t -i:"${port}" 2>/dev/null || true)

  if [ -n "$pid" ]; then
    echo "Found lingering ${service_name} process (pid: ${pid}) on port ${port}. Terminating..."
    # 强制杀死，因为这很可能是一个僵尸进程
    kill -9 "${pid}" 2>/dev/null || true
    # 等待一小段时间确保进程已退出
    sleep 0.5
    echo "Process ${pid} terminated."
  else
    echo "Port ${port} is clear."
  fi
}

build_and_start_services() {
  echo "Building auth and gateway services..."
  mkdir -p bin logs run || true
  go build -o bin/auth ./cmd/auth
  go build -o bin/user ./cmd/user
  go build -o bin/gateway ./cmd/gateway

  echo "Starting auth service in background..."
  nohup bin/auth > logs/auth.log 2>&1 & echo $! > run/auth.pid
  echo "Starting gateway service in background..."
  nohup bin/gateway > logs/gateway.log 2>&1 & echo $! > run/gateway.pid
  echo "Starting user service in background..."
  nohup bin/user > logs/user.log 2>&1 & echo $! > run/user.pid

}

stop_services() {
  echo "Stopping services managed by PID files..."
  for service in auth gateway; do
    local pid_file="run/${service}.pid"
    if [ ! -f "$pid_file" ]; then
      continue
    fi

    local pid
    pid=$(cat "$pid_file" 2>/dev/null || true)
    rm -f "$pid_file"

    if [ -z "$pid" ]; then
      echo "PID file for $service was empty. Cleaned up."
      continue
    fi

    if ps -p "$pid" > /dev/null; then
      echo "Stopping $service (pid: $pid)..."
      kill "$pid" 2>/dev/null || true
      sleep 0.5
      if ps -p "$pid" > /dev/null; then
        kill -9 "$pid" 2>/dev/null || true
      fi
      echo "$service (pid: $pid) stopped."
    else
      echo "Process for $service (pid: $pid) not found. PID file was stale."
    fi
  done

  # 双重保险：按端口号强制清理
  echo "Performing aggressive cleanup by port..."
  kill_process_on_port 50051 "auth"
  kill_process_on_port 50052 "user" # 假设 user 服务也可能在运行
  kill_process_on_port 8080 "gateway"

  echo "Service cleanup complete."
}


wait_for_gateway() {
  echo "Waiting for gateway HTTP /health on http://127.0.0.1:8080/health..."
  for i in {1..30}; do
    if curl -sS http://127.0.0.1:8080/health >/dev/null 2>&1; then
      echo "Gateway is ready"
      return 0
    fi
    sleep 1
  done
  echo "Gateway did not become ready" >&2
  return 1
}

http_register() {
  local user="$1"
  local pass="$2"
  echo "Registering user $user"
  curl -s -X POST -H 'Content-Type: application/json' -d "{\"username\":\"${user}\",\"password\":\"${pass}\"}" http://127.0.0.1:8080/api/v1/auth/register | jq || true
}

http_login() {
  local user="$1"
  local pass="$2"
  echo "Logging in user $user"
  curl -s -X POST -H 'Content-Type: application/json' -d "{\"username\":\"${user}\",\"password\":\"${pass}\",\"device_id\":101}" http://127.0.0.1:8080/api/v1/auth/login | jq || true
}

run_comprehensive_tests() {
  echo "Running comprehensive API tests..."
  if command -v "$ROOT_DIR/scripts/test_api.sh" >/dev/null 2>&1; then
    "$ROOT_DIR/scripts/test_api.sh"
  else
    echo "test_api.sh not found, running basic tests..."
    http_register "e2e_user" "password"
    http_login "e2e_user" "password"
  fi
}

run_flow() {
  if [ "$VERBOSE" -eq 1 ]; then
    echo "--- Running Pre-flight Cleanup ---"
    stop_services
    stop_containers
    echo "--- Pre-flight Cleanup Complete ---"

    start_containers
    sleep 2
    build_and_start_services

    if ! wait_for_gateway; then
      echo "Gateway not ready, aborting" >&2
      stop_services
      stop_containers
      exit 1
    fi

    run_comprehensive_tests

    stop_services
    stop_containers
  else
    # 静默模式：将详细输出写入日志文件
    echo "[e2e] $(date '+%F %T') Starting run flow (quiet). Logs -> $LOG_FILE"
    echo "--- Running Pre-flight Cleanup ---" >>"$LOG_FILE"
    stop_services >>"$LOG_FILE" 2>&1
    stop_containers >>"$LOG_FILE" 2>&1
    echo "--- Pre-flight Cleanup Complete ---" >>"$LOG_FILE"

    start_containers >>"$LOG_FILE" 2>&1
    sleep 2
    build_and_start_services >>"$LOG_FILE" 2>&1

    if ! wait_for_gateway >>"$LOG_FILE" 2>&1; then
      echo "Gateway not ready, aborting (see $LOG_FILE)" >&2
      stop_services >>"$LOG_FILE" 2>&1 || true
      stop_containers >>"$LOG_FILE" 2>&1 || true
      exit 1
    fi

    # 测试输出通常需要在终端查看，这里保持在终端显示
    run_comprehensive_tests

    stop_services >>"$LOG_FILE" 2>&1 || true
    stop_containers >>"$LOG_FILE" 2>&1 || true
  fi
}

case "$cmd" in
  start-containers)
    if [ "$VERBOSE" -eq 1 ]; then start_containers; else start_containers >>"$LOG_FILE" 2>&1; fi
    ;;
  stop-containers)
    if [ "$VERBOSE" -eq 1 ]; then stop_containers; else stop_containers >>"$LOG_FILE" 2>&1; fi
    ;;
  start-services)
    if [ "$VERBOSE" -eq 1 ]; then build_and_start_services; else build_and_start_services >>"$LOG_FILE" 2>&1; fi
    ;;
  stop-services)
    if [ "$VERBOSE" -eq 1 ]; then stop_services; else stop_services >>"$LOG_FILE" 2>&1; fi
    ;;
  run-tests)
    run_comprehensive_tests
    ;;
  run)
    run_flow
    ;;
  *)
    echo "Usage: $0 {start-containers|stop-containers|start-services|stop-services|run-tests|run} [ -v|--verbose | -q|--quiet ] [ --log-file <path> ]"
    echo "  run - start containers, build & start auth+gateway, wait for gateway, run comprehensive API tests, then cleanup"
    echo "  run-tests - run comprehensive API tests (assumes services are already running)"
    echo "  -v/--verbose to print detailed logs to terminal; default is quiet (logs to $LOG_FILE)"
    exit 2
    ;;
esac
