#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

cmd="${1:-}" || true
shift || true

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
  go build -o bin/gateway ./cmd/gateway

  echo "Starting auth service in background..."
  nohup bin/auth > logs/auth.log 2>&1 & echo $! > run/auth.pid
  echo "Starting gateway service in background..."
  nohup bin/gateway > logs/gateway.log 2>&1 & echo $! > run/gateway.pid
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
  # 关键改动：在所有操作开始前，先执行一次彻底的清理
  echo "--- Running Pre-flight Cleanup ---"
  stop_services
  stop_containers
  echo "--- Pre-flight Cleanup Complete ---"

  start_containers
  # give containers time to start
  sleep 2
  build_and_start_services
  # wait for gateway
  if ! wait_for_gateway; then
    echo "Gateway not ready, aborting" >&2
    stop_services
    stop_containers
    exit 1
  fi

  # run comprehensive api tests
  run_comprehensive_tests

  # cleanup
  stop_services
  stop_containers
}

case "$cmd" in
  start-containers)
    start_containers
    ;;
  stop-containers)
    stop_containers
    ;;
  start-services)
    build_and_start_services
    ;;
  stop-services)
    stop_services
    ;;
  run-tests)
    run_comprehensive_tests
    ;;
  run)
    run_flow
    ;;
  *)
    echo "Usage: $0 {start-containers|stop-containers|start-services|stop-services|run-tests|run}"
    echo "  run - start containers, build & start auth+gateway, wait for gateway, run comprehensive API tests, then cleanup"
    echo "  run-tests - run comprehensive API tests (assumes services are already running)"
    exit 2
    ;;
esac
