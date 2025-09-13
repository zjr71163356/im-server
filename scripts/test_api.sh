#!/usr/bin/env bash
# API Test Script for im-server grpc-gateway
# Comprehensive testing of all auth and user service endpoints
# Features:
# - Dynamic parameter extraction from login responses
# - Token verification using real tokens from login
# - Comprehensive error handling and edge case testing
# - Colored output with detailed test results

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

# Configuration
BASE_URL="http://localhost:8080"
TEST_USER="testuser_$(date +%s)"
TEST_PASS="testpass123"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_header() {
    echo ""
    echo "=========================================="
    echo "$1"
    echo "=========================================="
}

# Test result tracking
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
test_api() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    local description="$4"
    local expected_status="${5:-200}"
    local return_response="${6:-false}"

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    if [ "$return_response" = "true" ]; then
        >&2 echo ""
        >&2 print_info "测试: $description"
        >&2 echo "请求: $method $BASE_URL$endpoint"
        if [ -n "$data" ]; then
            >&2 echo "数据: $data"
        fi
    else
        echo ""
        print_info "测试: $description"
        echo "请求: $method $BASE_URL$endpoint"
        if [ -n "$data" ]; then
            echo "数据: $data"
        fi
    fi

    # Make the request
    if [ "$method" = "POST" ]; then
        response=$(curl -s -X POST "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data" \
            -w "HTTPSTATUS:%{http_code}")
    else
        response=$(curl -s -X GET "$BASE_URL$endpoint" \
            -w "HTTPSTATUS:%{http_code}")
    fi

    # Parse response
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')

    if [ "$return_response" = "true" ]; then
        >&2 echo "状态码: $http_code"
        >&2 echo "响应: [内容已返回给调用者]"
    else
        echo "状态码: $http_code"
        echo "响应: $body"
    fi

    # Check result
    if [ "$http_code" = "$expected_status" ]; then
        if [ "$return_response" = "true" ]; then
            >&2 print_success "测试通过"
            # 仅输出纯 JSON 到 stdout
            printf "%s" "$body"
        else
            print_success "测试通过"
        fi
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        if [ "$return_response" = "true" ]; then
            >&2 print_error "测试失败 (期望状态码: $expected_status, 实际: $http_code)"
            # 仍输出响应体，便于上层决定如何处理
            printf "%s" "$body"
        else
            print_error "测试失败 (期望状态码: $expected_status, 实际: $http_code)"
        fi
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Check if gateway is running
check_gateway() {
    print_info "检查 grpc-gateway 服务状态..."
    if curl -s "$BASE_URL/health" >/dev/null 2>&1; then
        print_success "Gateway 服务正在运行"
        return 0
    else
        print_error "Gateway 服务未运行，请先启动服务"
        print_info "启动命令: go run ./cmd/gateway"
        exit 1
    fi
}

# --------------------
# API 分组辅助函数
# --------------------

# 统一的登录（只执行登录，返回 JSON 响应体）
do_login() {
    print_info "执行用户登录（仅登录，返回响应）..."
    test_api "POST" "/api/v1/auth/login" \
        "{\"username\": \"$TEST_USER\", \"password\": \"$TEST_PASS\", \"device_id\": 12345}" \
        "用户登录" \
        "200" \
        "true"
}

# 解析登录响应（接受 JSON 响应作为参数），设置全局变量 user_id 和 token
parse_login_response() {
    local resp="$1"

    # 首选 jq 多路径解析（同时适配驼峰/下划线以及常见 data/result 包裹）
    if command -v jq >/dev/null 2>&1 && [ -n "$resp" ]; then
        user_id=$(echo "$resp" | jq -r '(.userId // .user_id // .data.userId // .data.user_id // .result.userId // .result.user_id) // empty' 2>/dev/null || echo "")
        token=$(echo "$resp"   | jq -r '(.token  // .data.token  // .result.token) // empty' 2>/dev/null || echo "")
    else
        user_id=""; token=""
    fi

    # 若 jq 未解析到，再做正则后备解析（宽松匹配）
    if [ -z "${user_id:-}" ]; then
        user_id=$(echo "$resp" | grep -oE '"user(Id|_id)"[[:space:]]*:[[:space:]]*"?[0-9]+' | grep -oE '[0-9]+' | head -1 || true)
    fi
    if [ -z "${token:-}" ]; then
        token=$(echo "$resp" | sed -nE 's/.*"token"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/p' | head -1 || true)
    fi

    # 输出解析结果或告警
    if [ -n "${user_id:-}" ] && [ -n "${token:-}" ] && [ "$user_id" != "null" ] && [ "$token" != "null" ]; then
        print_info "成功从登录响应中获取: user_id=$user_id, token=${token:0:20}..."
    else
        # 调试输出响应摘要（stderr），帮助定位问题
        >&2 print_warning "无法从登录响应中解析 userId 或 token，使用默认值"
        >&2 echo "登录响应摘要: $(echo "$resp" | tr -d '\n' | head -c 200)"
        user_id="10000"; token="mock_token_123"
    fi
}

# 分组：Auth.Register
group_auth_register() {
    print_header "[Group] AUTH - Register"
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"$TEST_USER\", \"password\": \"$TEST_PASS\"}" \
        "用户注册" "200"
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"$TEST_USER\", \"password\": \"$TEST_PASS\"}" \
        "重复用户名注册 (预期失败)" "400"
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"\"}" \
        "无效注册请求 (缺少密码)" "400"
}

# 分组：Auth.Login
group_auth_login() {
    print_header "[Group] AUTH - Login"
    # 执行登录并解析响应
    login_resp=$(do_login)
    parse_login_response "$login_resp"

    test_api "POST" "/api/v1/auth/login" \
        "{\"username\": \"$TEST_USER\", \"password\": \"wrongpass\", \"device_id\": 12345}" \
        "错误密码登录 (预期失败)" "401"
}

# 分组：Auth.Verify
group_auth_verify() {
    print_header "[Group] AUTH - Verify"
    # 若未登录过，先登录取 token
    if [ -z "${user_id:-}" ] || [ -z "${token:-}" ]; then
        login_resp=$(do_login)
        parse_login_response "$login_resp"
    fi
    test_api "POST" "/api/v1/auth/verify" \
        "{\"user_id\": $user_id, \"device_id\": 12345, \"token\": \"$token\"}" \
        "权限校验" "200"
}

# 分组：User.Search
group_user_search() {
    print_header "[Group] USER - Search"
    test_api "POST" "/api/v1/user/search" \
        "{\"keyword\": \"$TEST_USER\", \"page\": 1, \"page_size\": 10}" \
        "用户搜索 (使用刚注册用户名)" "200"
    test_api "POST" "/api/v1/user/search" \
        "{\"keyword\": \"\", \"page\": 1, \"page_size\": 5}" \
        "空关键字搜索" "200"
    test_api "POST" "/api/v1/user/search" \
        "{\"keyword\": \"$TEST_USER\", \"page\": 2, \"page_size\": 20}" \
        "用户搜索 (分页测试，使用刚注册用户名)" "200"
    test_api "POST" "/api/v1/user/search" \
        "{\"keyword\": \"not_exist_user_$(date +%s)\", \"page\": 1, \"page_size\": 5}" \
        "用户搜索 (不存在的用户名，预期空结果)" "200"
}

# --------------------
# 重构 run_tests，按 API 分组运行
# --------------------

# Run all tests
run_tests() {
    print_header "grpc-gateway API 完整测试"
    print_info "测试服务器: $BASE_URL"
    print_info "测试用户: $TEST_USER"
    check_gateway

    # 分组执行
    group_auth_register
    group_auth_login
    group_auth_verify

    group_user_search

    print_header "边界测试"
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"$(printf 'a%.0s' {1..100})\", \"password\": \"test\"}" \
        "超长用户名注册" "400"
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"\", \"password\": \"test\"}" \
        "空用户名注册" "400"
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"test\", \"password\":" \
        "格式错误的 JSON" "400"
}

# Print test summary
print_summary() {
    print_header "测试结果摘要"
    echo "总测试数: $TESTS_TOTAL"
    echo "通过: $TESTS_PASSED"
    echo "失败: $TESTS_FAILED"
    
    if [ "$TESTS_FAILED" -eq 0 ]; then
        print_success "所有测试通过! 🎉"
        exit 0
    else
        print_error "有 $TESTS_FAILED 个测试失败"
        exit 1
    fi
}

# Print usage
usage() {
    echo "用法: $0 [命令]"
    echo ""
    echo "功能特性:"
    echo "  • 动态参数提取: Token Verification 测试使用从登录响应中获取的真实参数"
    echo "  • 智能错误处理: 自动检测 jq 可用性并提供降级方案"
    echo "  • 彩色输出: 清晰的测试结果显示"
    echo "  • 边界测试: 包含各种异常情况和边界条件的测试"
    echo ""
    echo "命令:"
    echo "  test               - 运行所有 API 测试 (默认)"
    echo "  auth               - 运行所有 Auth 分组测试"
    echo "  user               - 运行所有 User 分组测试"
    echo "  group <name>       - 仅运行指定分组 (auth:register | auth:login | auth:verify | user:search)"
    echo "  help               - 显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                      # 运行所有测试"
    echo "  $0 group auth:login     # 只测试登录相关 API"
    echo "  $0 group user:search    # 只测试用户搜索 API"
}

# Main script
case "${1:-test}" in
    "test"|"")
        run_tests; print_summary ;;
    "auth")
        check_gateway
        group_auth_register
        group_auth_login
        group_auth_verify
        print_summary ;;
    "user")
        check_gateway
        group_user_search
        print_summary ;;
    "group")
        check_gateway
        name="${2:-}"
        case "$name" in
            "auth:register") group_auth_register ;;
            "auth:login")    group_auth_login ;;
            "auth:verify")   group_auth_verify ;;
            "user:search")   group_user_search ;;
            *) echo "未知分组: $name"; usage; exit 1 ;;
        esac
        print_summary ;;
    "help"|"-h"|"--help")
        usage ;;
    *)
        echo "未知命令: $1"; usage; exit 1 ;;
 esac
