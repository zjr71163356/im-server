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
  echo "Building auth, user, friend, and gateway services..."
  mkdir -p bin logs run || true
  go build -o bin/auth ./cmd/auth
  go build -o bin/user ./cmd/user
  go build -o bin/friend ./cmd/friend
  go build -o bin/gateway ./cmd/gateway

  echo "Starting auth service in background..."
  nohup bin/auth > logs/auth.log 2>&1 & echo $! > run/auth.pid
  echo "Starting user service in background..."
  nohup bin/user > logs/user.log 2>&1 & echo $! > run/user.pid
  echo "Starting friend service in background..."
  nohup bin/friend > logs/friend.log 2>&1 & echo $! > run/friend.pid
  echo "Starting gateway service in background..."
  nohup bin/gateway > logs/gateway.log 2>&1 & echo $! > run/gateway.pid

}

stop_services() {
  echo "Stopping services managed by PID files..."
  for service in auth user friend gateway; do
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
  kill_process_on_port 50052 "user"
  kill_process_on_port 50053 "friend"
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

# 新增：等待所有微服务就绪
wait_for_all_services() {
  echo "Waiting for all services to be ready..."
  
  # 等待gateway（包含健康检查端点）
  if ! wait_for_gateway; then
    return 1
  fi
  
  # 额外等待时间，确保所有gRPC服务完全启动
  echo "Waiting additional time for gRPC services to fully initialize..."
  sleep 3
  
  # 简单的连通性测试：尝试注册一个测试用户
  echo "Testing service connectivity with a simple user registration..."
  local test_user="connectivity_test_$(date +%s)"
  local test_response
  test_response=$(curl -s -X POST -H 'Content-Type: application/json' \
    -d "{\"username\":\"$test_user\",\"password\":\"testpass\"}" \
    http://127.0.0.1:8080/api/v1/auth/register 2>/dev/null || echo "")
  
  if [ -n "$test_response" ]; then
    echo "Service connectivity test successful"
    return 0
  else
    echo "Service connectivity test failed" >&2
    return 1
  fi
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

# 新增：好友相关的HTTP测试函数
http_send_friend_request() {
  local token="$1"
  local recipient_id="$2"
  local message="$3"
  echo "Sending friend request to user $recipient_id"
  curl -s -X POST -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $token" \
    -d "{\"recipient_id\":$recipient_id,\"message\":\"$message\"}" \
    http://127.0.0.1:8080/api/v1/friend/request | jq || true
}

http_get_received_friend_requests() {
  local token="$1"
  echo "Getting received friend requests"
  curl -s -X GET -H "Authorization: Bearer $token" \
    "http://127.0.0.1:8080/api/v1/friend/requests/received?page=1&page_size=10" | jq || true
}

http_get_sent_friend_requests() {
  local token="$1"
  echo "Getting sent friend requests"
  curl -s -X GET -H "Authorization: Bearer $token" \
    "http://127.0.0.1:8080/api/v1/friend/requests/sent?page=1&page_size=10" | jq || true
}

http_handle_friend_request() {
  local token="$1"
  local request_id="$2"
  local action="$3"  # 1=accept, 2=reject, 3=ignore
  echo "Handling friend request $request_id with action $action"
  curl -s -X PUT -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $token" \
    -d "{\"action\":$action}" \
    "http://127.0.0.1:8080/api/v1/friend/request/$request_id" | jq || true
}

http_get_friend_list() {
  local token="$1"
  echo "Getting friend list"
  curl -s -X GET -H "Authorization: Bearer $token" \
    "http://127.0.0.1:8080/api/v1/friend/list?page=1&page_size=10" | jq || true
}

http_search_user() {
  local token="$1"
  local keyword="$2"
  echo "Searching for user with keyword: $keyword"
  curl -s -X POST -H 'Content-Type: application/json' \
    -H "Authorization: Bearer $token" \
    -d "{\"keyword\":\"$keyword\",\"page\":1,\"page_size\":10}" \
    http://127.0.0.1:8080/api/v1/user/search | jq || true
}

# 扩展的鉴权测试
run_auth_security_tests() {
  echo "=== Running Authentication Security Tests ==="
  
  echo "Test 1: Request without Authorization header"
  curl -s -X POST -H 'Content-Type: application/json' \
    -d '{"recipient_id":10001,"message":"No auth"}' \
    http://127.0.0.1:8080/api/v1/friend/request || true
  
  echo "Test 2: Request with invalid token format"
  curl -s -X POST -H 'Content-Type: application/json' \
    -H "Authorization: InvalidFormat" \
    -d '{"recipient_id":10001,"message":"Invalid format"}' \
    http://127.0.0.1:8080/api/v1/friend/request || true
  
  echo "Test 3: Request with malformed Bearer token"
  curl -s -X POST -H 'Content-Type: application/json' \
    -H "Authorization: Bearer malformed.jwt.token" \
    -d '{"recipient_id":10001,"message":"Malformed token"}' \
    http://127.0.0.1:8080/api/v1/friend/request || true
  
  echo "=== Authentication Security Tests Complete ==="
}

run_comprehensive_tests() {
  echo "Running comprehensive API tests..."
  
  # 1. 运行完整的API测试套件（如果存在）
  if command -v "$ROOT_DIR/scripts/test_api.sh" >/dev/null 2>&1; then
    echo "=== Running Full API Test Suite ==="
    "$ROOT_DIR/scripts/test_api.sh"
  else
    echo "test_api.sh not found, running basic tests..."
    http_register "e2e_user" "password"
    http_login "e2e_user" "password"
  fi
  
  # 2. 运行鉴权安全测试
  run_auth_security_tests
  
  echo "=== All Comprehensive Tests Complete ==="
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

    if ! wait_for_all_services; then
      echo "Services not ready, aborting" >&2
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

    if ! wait_for_all_services >>"$LOG_FILE" 2>&1; then
      echo "Services not ready, aborting (see $LOG_FILE)" >&2
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
  run-auth-tests)
    run_auth_security_tests
    ;;
  run-api-tests)
    if command -v "$ROOT_DIR/scripts/test_api.sh" >/dev/null 2>&1; then
      "$ROOT_DIR/scripts/test_api.sh"
    else
      echo "test_api.sh not found"
      exit 1
    fi
    ;;
  run)
    run_flow
    ;;
  *)
    echo "Usage: $0 {start-containers|stop-containers|start-services|stop-services|run-tests|run-auth-tests|run-api-tests|run} [ -v|--verbose | -q|--quiet ] [ --log-file <path> ]"
    echo ""
    echo "Commands:"
    echo "  run                - Full E2E: start containers, build & start all services, wait for gateway, run comprehensive tests, then cleanup"
    echo "  run-tests          - Run all comprehensive tests (assumes services are already running)"
    echo "  run-api-tests      - Run API test suite only (scripts/test_api.sh)"
    echo "  run-auth-tests     - Run authentication security tests"
    echo "  start-containers   - Start Redis and MySQL containers"
    echo "  stop-containers    - Stop Redis and MySQL containers"
    echo "  start-services     - Build and start all microservices (auth, user, friend, gateway)"
    echo "  stop-services      - Stop all microservices"
    echo ""
    echo "Options:"
    echo "  -v/--verbose       - Print detailed logs to terminal"
    echo "  -q/--quiet         - Quiet mode (default), logs to $LOG_FILE"
    echo "  --log-file <path>  - Specify custom log file path"
    echo ""
    echo "Examples:"
    echo "  $0 run -v                    # Full E2E test with verbose output"
    echo "  $0 start-services           # Start services only"
    echo "  $0 run-tests                # Run tests against running services"
    exit 2
    ;;
esac
