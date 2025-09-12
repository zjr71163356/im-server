#!/usr/bin/env bash
# API Test Script for im-server grpc-gateway
# Comprehensive testing of all auth and user service endpoints

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

    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    echo ""
    print_info "测试: $description"
    echo "请求: $method $BASE_URL$endpoint"
    
    if [ -n "$data" ]; then
        echo "数据: $data"
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

    echo "状态码: $http_code"
    echo "响应: $body"

    # Check result
    if [ "$http_code" = "$expected_status" ]; then
        print_success "测试通过"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        print_error "测试失败 (期望状态码: $expected_status, 实际: $http_code)"
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

# Run all tests
run_tests() {
    print_header "grpc-gateway API 完整测试"
    print_info "测试服务器: $BASE_URL"
    print_info "测试用户: $TEST_USER"
    
    check_gateway

    print_header "AUTH 服务测试"

    # Test 1: User Registration
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"$TEST_USER\", \"password\": \"$TEST_PASS\"}" \
        "用户注册" \
        "200"

    # Test 2: User Registration with same username (should fail)
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"$TEST_USER\", \"password\": \"$TEST_PASS\"}" \
        "重复用户名注册 (预期失败)" \
        "400"

    # Test 3: User Login
    test_api "POST" "/api/v1/auth/login" \
        "{\"username\": \"$TEST_USER\", \"password\": \"$TEST_PASS\", \"device_id\": 12345}" \
        "用户登录" \
        "200"

    # Test 4: User Login with wrong password
    test_api "POST" "/api/v1/auth/login" \
        "{\"username\": \"$TEST_USER\", \"password\": \"wrongpass\", \"device_id\": 12345}" \
        "错误密码登录 (预期失败)" \
        "401"

    # Test 5: Token Verification
    test_api "POST" "/api/v1/auth/verify" \
        "{\"user_id\": 10000, \"device_id\": 12345, \"token\": \"mock_token_123\"}" \
        "权限校验" \
        "200"

    # Test 6: Invalid Registration (missing fields)
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"\"}" \
        "无效注册请求 (缺少密码)" \
        "400"

    print_header "USER 服务测试"

    # Test 7: User Search
    test_api "POST" "/api/v1/user/search" \
        "{\"keyword\": \"test\", \"page\": 1, \"page_size\": 10}" \
        "用户搜索" \
        "200"

    # Test 8: User Search with empty keyword
    test_api "POST" "/api/v1/user/search" \
        "{\"keyword\": \"\", \"page\": 1, \"page_size\": 5}" \
        "空关键字搜索" \
        "200"

    # Test 9: User Search with pagination
    test_api "POST" "/api/v1/user/search" \
        "{\"keyword\": \"user\", \"page\": 2, \"page_size\": 20}" \
        "用户搜索 (分页测试)" \
        "200"

    # Test 10: User Search with invalid pagination
    test_api "POST" "/api/v1/user/search" \
        "{\"keyword\": \"test\", \"page\": 0, \"page_size\": 0}" \
        "无效分页参数搜索" \
        "200"

    # Test 11: User Search with large page size
    test_api "POST" "/api/v1/user/search" \
        "{\"keyword\": \"test\", \"page\": 1, \"page_size\": 100}" \
        "大分页搜索" \
        "200"

    print_header "边界测试"

    # Test 12: Register with very long username
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"$(printf 'a%.0s' {1..100})\", \"password\": \"test\"}" \
        "超长用户名注册" \
        "400"

    # Test 13: Register with empty username
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"\", \"password\": \"test\"}" \
        "空用户名注册" \
        "400"

    # Test 14: Malformed JSON
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"test\", \"password\":" \
        "格式错误的 JSON" \
        "400"
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
    echo "命令:"
    echo "  test      - 运行所有 API 测试 (默认)"
    echo "  auth      - 只运行 auth 服务测试"
    echo "  user      - 只运行 user 服务测试"
    echo "  help      - 显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0          # 运行所有测试"
    echo "  $0 test     # 运行所有测试"
    echo "  $0 auth     # 只测试认证服务"
}

# Main script
case "${1:-test}" in
    "test"|"")
        run_tests
        print_summary
        ;;
    "auth")
        check_gateway
        print_header "AUTH 服务测试"
        # Only run auth tests (tests 1-6)
        test_api "POST" "/api/v1/auth/register" \
            "{\"username\": \"$TEST_USER\", \"password\": \"$TEST_PASS\"}" \
            "用户注册" "200"
        test_api "POST" "/api/v1/auth/login" \
            "{\"username\": \"$TEST_USER\", \"password\": \"$TEST_PASS\", \"device_id\": 12345}" \
            "用户登录" "200"
        test_api "POST" "/api/v1/auth/verify" \
            "{\"user_id\": 10000, \"device_id\": 12345, \"token\": \"mock_token_123\"}" \
            "权限校验" "200"
        print_summary
        ;;
    "user")
        check_gateway
        print_header "USER 服务测试"
        # Only run user tests (tests 7-11)
        test_api "POST" "/api/v1/user/search" \
            "{\"keyword\": \"test\", \"page\": 1, \"page_size\": 10}" \
            "用户搜索" "200"
        test_api "POST" "/api/v1/user/search" \
            "{\"keyword\": \"\", \"page\": 1, \"page_size\": 5}" \
            "空关键字搜索" "200"
        print_summary
        ;;
    "help"|"-h"|"--help")
        usage
        ;;
    *)
        echo "未知命令: $1"
        usage
        exit 1
        ;;
esac
