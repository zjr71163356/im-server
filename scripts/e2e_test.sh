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
  echo "Stopping auth and gateway services (if started by this script)..."
  if [ -f run/auth.pid ]; then
    PID=$(cat run/auth.pid)
    kill "$PID" || true
    rm -f run/auth.pid
    echo "Stopped auth (pid $PID)"
  fi
  if [ -f run/gateway.pid ]; then
    PID=$(cat run/gateway.pid)
    kill "$PID" || true
    rm -f run/gateway.pid
    echo "Stopped gateway (pid $PID)"
  fi
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

run_flow() {
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

  # run http tests
  http_register "e2e_user" "password"
  http_login "e2e_user" "password"

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
  run)
    run_flow
    ;;
  *)
    echo "Usage: $0 {start-containers|stop-containers|start-services|stop-services|run}"
    echo "  run - start containers, build & start auth+gateway, wait for gateway, run HTTP register/login tests, then cleanup"
    exit 2
    ;;
esac
