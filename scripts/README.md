# Scripts Directory

This directory contains testing and utility scripts for the im-server project.

## Available Scripts

### test_api.sh

Comprehensive API testing script for all auth and user service endpoints via grpc-gateway.

**Usage:**

```bash
# Run all tests
./scripts/test_api.sh

# Run only auth service tests
./scripts/test_api.sh auth

# Run only user service tests
./scripts/test_api.sh user

# Show help
./scripts/test_api.sh help
```

**Features:**

- ✅ Complete auth service testing (register, login, verify)
- ✅ Complete user service testing (search with various parameters)
- ✅ Edge case testing (invalid inputs, boundary conditions)
- ✅ Colored output with success/failure indicators
- ✅ Test result summary and statistics
- ✅ Service availability checking

**Test Coverage:**

- User registration (valid and invalid cases)
- User login (correct and incorrect credentials)
- Token verification
- User search (with different keywords and pagination)
- Boundary testing (empty fields, long inputs, malformed JSON)

### e2e_test.sh

End-to-end testing script that manages the full testing lifecycle.

**Usage:**

```bash
# Full e2e flow: start containers, services, run tests, cleanup
./scripts/e2e_test.sh run

# Run comprehensive API tests (assumes services running)
./scripts/e2e_test.sh run-tests

# Start only containers
./scripts/e2e_test.sh start-containers

# Start only services
./scripts/e2e_test.sh start-services

# Stop services
./scripts/e2e_test.sh stop-services

# Stop containers
./scripts/e2e_test.sh stop-containers
```

**What it does:**

1. Starts Redis and MySQL containers
2. Builds and starts auth and gateway services
3. Waits for services to be ready
4. Runs comprehensive API tests via `test_api.sh`
5. Cleans up services and containers

## Prerequisites

### Required Services

- **Auth Service**: Must be running on localhost:50051
- **User Service**: Must be running on localhost:50052
- **Gateway Service**: Must be running on localhost:8080

### Required Tools

- `curl` - for HTTP requests
- `jq` - for JSON formatting (optional, graceful fallback)
- Docker - for database containers

### Database Setup

The scripts assume MySQL and Redis are available:

- **MySQL**: localhost:3307 with password `azsx0123456`
- **Redis**: localhost:6379

## Testing Workflow

### Development Testing

```bash
# Start services manually
go run ./cmd/auth &
go run ./cmd/user &
go run ./cmd/gateway &

# Run API tests
./scripts/test_api.sh
```

### CI/CD Testing

```bash
# Full automated flow
./scripts/e2e_test.sh run
```

### Service-Specific Testing

```bash
# Test only auth endpoints
./scripts/test_api.sh auth

# Test only user endpoints
./scripts/test_api.sh user
```

## Expected Output

### Successful Test Run

```
==========================================
grpc-gateway API 完整测试
==========================================
ℹ 测试服务器: http://localhost:8080
ℹ 测试用户: testuser_1640995200

==========================================
AUTH 服务测试
==========================================

ℹ 测试: 用户注册
请求: POST http://localhost:8080/api/v1/auth/register
数据: {"username": "testuser_1640995200", "password": "testpass123"}
状态码: 200
响应: {"userId":"10000", "message":"注册成功"}
✓ 测试通过

...

==========================================
测试结果摘要
==========================================
总测试数: 14
通过: 14
失败: 0
✓ 所有测试通过! 🎉
```

## Troubleshooting

### Gateway Not Running

```
✗ Gateway 服务未运行，请先启动服务
ℹ 启动命令: go run ./cmd/gateway
```

### Service Connection Issues

Check if the gRPC services are running:

```bash
# Check auth service
curl -X POST http://localhost:8080/api/v1/auth/register -d '{"username":"test","password":"test"}'

# If you get connection errors, start the services:
go run ./cmd/auth &
go run ./cmd/user &
```

### Database Issues

```bash
# Start required containers
./scripts/e2e_test.sh start-containers

# Check container status
docker ps | grep -E "(redis|mysql)"
```
