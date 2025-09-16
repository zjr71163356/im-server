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
    elif [ "$method" = "PUT" ]; then
        response=$(curl -s -X PUT "$BASE_URL$endpoint" \
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
        >&2 echo "响应: $body"
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

# Test function with authentication
test_api_with_auth() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    local description="$4"
    local expected_status="${5:-200}"
    local return_response="${6:-false}"

    # 确保有有效的token
    if [ -z "${token:-}" ] || [ "${token:-}" = "mock_token_123" ]; then
        print_warning "需要有效token，正在执行登录..."
        login_resp=$(do_login)
        parse_login_response "$login_resp"
    fi

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

    # Make the request with Authorization header
    if [ "$method" = "POST" ]; then
        response=$(curl -s -X POST "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $token" \
            -d "$data" \
            -w "HTTPSTATUS:%{http_code}")
    elif [ "$method" = "GET" ]; then
        response=$(curl -s -X GET "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $token" \
            -w "HTTPSTATUS:%{http_code}")
    elif [ "$method" = "PUT" ]; then
        response=$(curl -s -X PUT "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $token" \
            -d "$data" \
            -w "HTTPSTATUS:%{http_code}")
    else
        response=$(curl -s -X "$method" "$BASE_URL$endpoint" \
            -H "Authorization: Bearer $token" \
            -w "HTTPSTATUS:%{http_code}")
    fi

    # Parse response (same as test_api function)
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')

    if [ "$return_response" = "true" ]; then
        >&2 echo "状态码: $http_code"
        >&2 echo "响应: $body"
    else
        echo "状态码: $http_code"
        echo "响应: $body"
    fi

    # Check result
    if [ "$http_code" = "$expected_status" ]; then
        if [ "$return_response" = "true" ]; then
            >&2 print_success "测试通过"
            printf "%s" "$body"
        else
            print_success "测试通过"
        fi
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        if [ "$return_response" = "true" ]; then
            >&2 print_error "测试失败 (期望状态码: $expected_status, 实际: $http_code)"
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

# Assert helper for user search: call API and validate users array
assert_search() {
    local data="$1"
    local description="$2"
    local expected_status="$3"
    local expect_type="$4" # "nonempty" | "empty" | "any" | "contains:<username>"

    # call test_api and capture body
    resp=$(test_api "POST" "/api/v1/user/search" "$data" "$description" "$expected_status" "true")

    # if expected non-empty or specific check, parse
    if command -v jq >/dev/null 2>&1; then
        users_len=$(echo "$resp" | jq -r '.users | length // 0' 2>/dev/null || echo "0")
        if [ "$expect_type" = "nonempty" ]; then
            if [ "$users_len" -le 0 ]; then
                print_error "断言失败: $description - 期望非空结果，实际 users_len=$users_len"
                TESTS_FAILED=$((TESTS_FAILED+1))
                return 1
            fi
        elif [ "$expect_type" = "empty" ]; then
            if [ "$users_len" -ne 0 ]; then
                print_error "断言失败: $description - 期望空结果，实际 users_len=$users_len"
                TESTS_FAILED=$((TESTS_FAILED+1))
                return 1
            fi
        elif [[ "$expect_type" == contains:* ]]; then
            local want=${expect_type#contains:}
            found=$(echo "$resp" | jq -r --arg w "$want" '.users[]?.username | select(. == $w) // empty' 2>/dev/null || true)
            if [ -z "$found" ]; then
                print_error "断言失败: $description - 期望包含 username=$want，但未找到"
                TESTS_FAILED=$((TESTS_FAILED+1))
                return 1
            fi
        fi
    else
        # no jq: best-effort regex checks
        if [ "$expect_type" = "nonempty" ]; then
            if ! echo "$resp" | grep -q '"users":\s*\['; then
                print_error "断言失败: $description - 期望非空结果，未找到 users 数组"
                TESTS_FAILED=$((TESTS_FAILED+1))
                return 1
            fi
        fi
    fi

    return 0
}

# 分组：User.Search
group_user_search() {
    print_header "[Group] USER - Search"
    # 使用刚注册用户名进行精确搜索并断言包含该用户名
    assert_search "{\"keyword\": \"$TEST_USER\", \"page\": 1, \"page_size\": 10}" "用户搜索 (使用刚注册用户名)" "200" "contains:$TEST_USER"

    # 空关键字请求应返回 400（接口约束）
    test_api "POST" "/api/v1/user/search" \
        "{\"keyword\": \"\", \"page\": 1, \"page_size\": 5}" \
        "空关键字搜索" "400"

    # 分页测试：至少返回非空结果
    assert_search "{\"keyword\": \"$TEST_USER\", \"page\": 1, \"page_size\": 20}" "用户搜索 (分页测试，使用刚注册用户名)" "200" "nonempty"

    # 另一个分页变体（保持与上面相同目的）
    assert_search "{\"keyword\": \"$TEST_USER\", \"page\": 1, \"page_size\": 20}" "用户搜索 (分页测试，使用刚注册用户名)" "200" "nonempty"

    # 搜不到用户，应返回空结果
    assert_search "{\"keyword\": \"not_exist_user_$(date +%s)\", \"page\": 1, \"page_size\": 5}" "用户搜索 (不存在的用户名，预期空结果)" "200" "empty"

    # 通用昵称搜索，期待返回用户列表（非空）
    assert_search "{\"keyword\": \"user\", \"page\": 1, \"page_size\": 5}" "昵称搜索,返回用户列表" "200" "nonempty"
}

# 新增：创建第二个测试用户的函数
create_second_user() {
    local second_user="friend_target_$(date +%s)"
    local second_pass="testpass456"
    
    print_info "创建第二个测试用户用于好友申请测试..."
    test_api "POST" "/api/v1/auth/register" \
        "{\"username\": \"$second_user\", \"password\": \"$second_pass\"}" \
        "注册第二个测试用户" "200"
    
    # 获取第二个用户的信息
    if command -v jq >/dev/null 2>&1; then
        # 搜索刚创建的用户获取其ID
        search_resp=$(test_api_with_auth "POST" "/api/v1/user/search" \
            "{\"keyword\": \"$second_user\", \"page\": 1, \"page_size\": 1}" \
            "搜索第二个用户获取ID" "200" "true")
        
        second_user_id=$(echo "$search_resp" | jq -r '.users[0].id // .users[0].user_id // empty' 2>/dev/null || echo "")
        if [ -z "$second_user_id" ] || [ "$second_user_id" = "null" ]; then
            second_user_id="10001"  # 默认值
            print_warning "无法获取第二个用户ID，使用默认值: $second_user_id"
        else
            print_info "第二个用户ID: $second_user_id"
        fi
    else
        second_user_id="10001"  # 无jq时的默认值
    fi
    
    echo "$second_user_id"  # 返回用户ID
}

# 分组：Friend.SendRequest - 发送好友申请
group_friend_send_request() {
    print_header "[Group] FRIEND - Send Request"
    
    # 创建第二个用户用于测试
    target_user_id=$(create_second_user)
    
    # 测试发送好友申请
    test_api_with_auth "POST" "/api/v1/friend/request" \
        "{\"recipient_id\": $target_user_id, \"message\": \"Hello, let's be friends!\"}" \
        "发送好友申请" "200"
    
    # 测试向自己发送好友申请（应该失败）
    test_api_with_auth "POST" "/api/v1/friend/request" \
        "{\"recipient_id\": $user_id, \"message\": \"Cannot send to myself\"}" \
        "向自己发送好友申请 (预期失败)" "400"
    
    # 测试重复发送好友申请（应该失败）
    test_api_with_auth "POST" "/api/v1/friend/request" \
        "{\"recipient_id\": $target_user_id, \"message\": \"Duplicate request\"}" \
        "重复发送好友申请 (预期失败)" "400"
    
    # 测试无效的recipient_id
    test_api_with_auth "POST" "/api/v1/friend/request" \
        "{\"recipient_id\": 99999, \"message\": \"Invalid user\"}" \
        "发送给不存在用户的好友申请 (预期失败)" "400"
    
    # 测试无效请求（缺少recipient_id）
    test_api_with_auth "POST" "/api/v1/friend/request" \
        "{\"message\": \"Missing recipient_id\"}" \
        "缺少recipient_id的好友申请 (预期失败)" "400"
}

# 分组：Friend.GetRequests - 查看好友申请列表
group_friend_get_requests() {
    print_header "[Group] FRIEND - Get Requests"
    
    # 测试获取收到的好友申请列表
    test_api_with_auth "GET" "/api/v1/friend/requests/received?page=1&page_size=10" \
        "" \
        "获取收到的好友申请列表" "200"
    
    # 测试获取发送的好友申请列表
    test_api_with_auth "GET" "/api/v1/friend/requests/sent?page=1&page_size=10" \
        "" \
        "获取发送的好友申请列表" "200"
    
    # 测试分页参数
    test_api_with_auth "GET" "/api/v1/friend/requests/received?page=2&page_size=5" \
        "" \
        "分页获取好友申请 (第2页)" "200"
    
    # 测试无效的分页参数
    test_api_with_auth "GET" "/api/v1/friend/requests/received?page=0&page_size=10" \
        "" \
        "无效分页参数 (page=0)" "400"
    
    test_api_with_auth "GET" "/api/v1/friend/requests/received?page=1&page_size=101" \
        "" \
        "无效分页参数 (page_size过大)" "400"
}

# 分组：Friend.HandleRequest - 处理好友申请
group_friend_handle_request() {
    print_header "[Group] FRIEND - Handle Request"
    
    # 首先获取一个好友申请ID用于测试
    print_info "获取好友申请ID用于测试..."
    requests_resp=$(test_api_with_auth "GET" "/api/v1/friend/requests/received?page=1&page_size=1" \
        "" \
        "获取好友申请ID" "200" "true")
    
    # 解析好友申请ID
    if command -v jq >/dev/null 2>&1 && [ -n "$requests_resp" ]; then
        request_id=$(echo "$requests_resp" | jq -r '.requests[0].id // .requests[0].request_id // empty' 2>/dev/null || echo "")
        if [ -z "$request_id" ] || [ "$request_id" = "null" ]; then
            request_id="1"  # 默认值
            print_warning "无法获取好友申请ID，使用默认值: $request_id"
        else
            print_info "使用好友申请ID: $request_id"
        fi
    else
        request_id="1"  # 无jq时的默认值
    fi
    
    # 测试同意好友申请
    test_api_with_auth "PUT" "/api/v1/friend/request/$request_id" \
        "{\"action\": 1}" \
        "同意好友申请" "200"
    
    # 创建新的好友申请用于拒绝测试
    target_user_id_2=$((10000 + $(date +%s) % 1000))
    test_api_with_auth "POST" "/api/v1/friend/request" \
        "{\"recipient_id\": $target_user_id_2, \"message\": \"For reject test\"}" \
        "创建用于拒绝测试的好友申请" "200"
    
    # 获取新创建的申请ID并拒绝
    new_request_id=$((request_id + 1))  # 简化处理
    test_api_with_auth "PUT" "/api/v1/friend/request/$new_request_id" \
        "{\"action\": 2}" \
        "拒绝好友申请" "200"
    
    # 测试忽略好友申请
    ignore_request_id=$((request_id + 2))
    test_api_with_auth "PUT" "/api/v1/friend/request/$ignore_request_id" \
        "{\"action\": 3}" \
        "忽略好友申请" "200"
    
    # 测试无效的申请ID
    test_api_with_auth "PUT" "/api/v1/friend/request/99999" \
        "{\"action\": 1}" \
        "处理不存在的好友申请 (预期失败)" "404"
    
    # 测试无效的action
    test_api_with_auth "PUT" "/api/v1/friend/request/$request_id" \
        "{\"action\": 999}" \
        "无效的处理动作 (预期失败)" "400"
    
    # 测试缺少action参数
    test_api_with_auth "PUT" "/api/v1/friend/request/$request_id" \
        "{}" \
        "缺少action参数 (预期失败)" "400"
    
    # 测试处理已经处理过的申请
    test_api_with_auth "PUT" "/api/v1/friend/request/$request_id" \
        "{\"action\": 1}" \
        "重复处理已处理的申请 (预期失败)" "400"
}

# 分组：Friend.GetFriendList - 获取好友列表
group_friend_list() {
    print_header "[Group] FRIEND - Get Friend List"
    
    # 测试获取好友列表
    test_api_with_auth "GET" "/api/v1/friend/list?page=1&page_size=10" \
        "" \
        "获取好友列表" "200"
    
    # 测试按分类获取好友列表
    test_api_with_auth "GET" "/api/v1/friend/list?category_id=1&page=1&page_size=10" \
        "" \
        "按分类获取好友列表" "200"
    
    # 测试分页参数
    test_api_with_auth "GET" "/api/v1/friend/list?page=2&page_size=5" \
        "" \
        "分页获取好友列表" "200"
    
    # 测试无效的分页参数
    test_api_with_auth "GET" "/api/v1/friend/list?page=0&page_size=10" \
        "" \
        "无效分页参数 (page=0)" "400"
}

# 分组：测试JWT鉴权中间件
group_auth_middleware() {
    print_header "[Group] AUTH - Middleware Testing"
    
    # 测试无Authorization header的请求
    test_api "POST" "/api/v1/friend/request" \
        "{\"recipient_id\": 10001, \"message\": \"No auth header\"}" \
        "无Authorization header的好友申请 (预期失败)" "401"
    
    # 测试无效的token格式
    response=$(curl -s -X POST "$BASE_URL/api/v1/friend/request" \
        -H "Content-Type: application/json" \
        -H "Authorization: InvalidTokenFormat" \
        -d "{\"recipient_id\": 10001, \"message\": \"Invalid token format\"}" \
        -w "HTTPSTATUS:%{http_code}")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    if [ "$http_code" = "401" ]; then
        print_success "无效token格式测试通过"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        print_error "无效token格式测试失败 (期望: 401, 实际: $http_code)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    
    # 测试过期的token
    response=$(curl -s -X POST "$BASE_URL/api/v1/friend/request" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer expired.jwt.token" \
        -d "{\"recipient_id\": 10001, \"message\": \"Expired token\"}" \
        -w "HTTPSTATUS:%{http_code}")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    if [ "$http_code" = "401" ]; then
        print_success "过期token测试通过"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        print_error "过期token测试失败 (期望: 401, 实际: $http_code)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
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

    # 新增的好友相关测试分组
    group_friend_send_request
    group_friend_get_requests
    group_friend_handle_request
    group_friend_list

    # 新增的鉴权中间件测试
    group_auth_middleware

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
    echo "  • JWT 鉴权测试: 测试JWT token的验证和用户身份解析"
    echo "  • 好友系统测试: 完整的好友申请、查看、处理流程测试"
    echo "  • 智能错误处理: 自动检测 jq 可用性并提供降级方案"
    echo "  • 彩色输出: 清晰的测试结果显示"
    echo "  • 边界测试: 包含各种异常情况和边界条件的测试"
    echo ""
    echo "命令:"
    echo "  test               - 运行所有 API 测试 (默认)"
    echo "  auth               - 运行所有 Auth 分组测试"
    echo "  user               - 运行所有 User 分组测试"
    echo "  friend             - 运行所有 Friend 分组测试"
    echo "  middleware         - 运行鉴权中间件测试"
    echo "  group <name>       - 仅运行指定分组"
    echo "  help               - 显示帮助信息"
    echo ""
    echo "分组名称:"
    echo "  auth:register      - 用户注册测试"
    echo "  auth:login         - 用户登录测试"
    echo "  auth:verify        - token验证测试"
    echo "  user:search        - 用户搜索测试"
    echo "  friend:send        - 发送好友申请测试"
    echo "  friend:requests    - 查看好友申请列表测试"
    echo "  friend:handle      - 处理好友申请测试"
    echo "  friend:list        - 获取好友列表测试"
    echo "  auth:middleware    - JWT鉴权中间件测试"
    echo ""
    echo "示例:"
    echo "  $0                      # 运行所有测试"
    echo "  $0 friend               # 运行所有好友相关测试"
    echo "  $0 group friend:send    # 只测试发送好友申请 API"
    echo "  $0 middleware           # 只测试鉴权中间件"
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
    "friend")
        check_gateway
        # 确保先有登录用户
        login_resp=$(do_login)
        parse_login_response "$login_resp"
        group_friend_send_request
        group_friend_get_requests
        group_friend_handle_request
        group_friend_list
        print_summary ;;
    "middleware")
        check_gateway
        group_auth_middleware
        print_summary ;;
    "group")
        check_gateway
        name="${2:-}"
        case "$name" in
            "auth:register") group_auth_register ;;
            "auth:login")    group_auth_login ;;
            "auth:verify")   group_auth_verify ;;
            "user:search")   group_user_search ;;
            "friend:send")   
                login_resp=$(do_login)
                parse_login_response "$login_resp"
                group_friend_send_request ;;
            "friend:requests")   
                login_resp=$(do_login)
                parse_login_response "$login_resp"
                group_friend_get_requests ;;
            "friend:handle") 
                login_resp=$(do_login)
                parse_login_response "$login_resp"
                group_friend_handle_request ;;
            "friend:list") 
                login_resp=$(do_login)
                parse_login_response "$login_resp"
                group_friend_list ;;
            "auth:middleware") group_auth_middleware ;;
            *) echo "未知分组: $name"; usage; exit 1 ;;
        esac
        print_summary ;;
    "help"|"-h"|"--help")
        usage ;;
    *)
        echo "未知命令: $1"; usage; exit 1 ;;
esac
