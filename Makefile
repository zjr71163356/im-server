# 查找项目中需要编译的 .proto 文件（排除vendor目录）
PROTO_FILES := $(shell find pkg/protocol/proto -name "*.proto")
DATABASE_URL := "mysql://root:azsx0123456@tcp(localhost:3307)/imserver?multiStatements=true"
DAO_PATH := "pkg/dao"


# 将 Go 的 bin 目录添加到 PATH
GOPATH := $(shell go env GOPATH)
export PATH := $(GOPATH)/bin:$(PATH)


getip:
	@echo "Getting local IP address..."
	@export LOCAL_IP=$$(hostname -I | awk '{print $$1}'); \
	echo "Local IP address is $$LOCAL_IP"
# 运行docker命令启动mysql
startdb:
	@echo "Starting MySQL Docker container..."
	docker start mysql_db


# 数据库迁移相关
migrate-up:
	@echo "Running database migrations up..."
	migrate -database $(DATABASE_URL) -path db/migrations up
	@echo "Database migrations up complete."

migrate-down:
	@echo "Running database migrations down..."
	migrate -database $(DATABASE_URL) -path db/migrations down
	@echo "Database migrations down complete."

# 创建新的迁移文件
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir db/migrations -seq $$name

# 生成代码相关
proto:
	@echo "Removing existing generated protobuf files..."
	@find . -type f -name "*.pb.go" -print -exec rm -f {} \;
	@find . -type f -name "*.pb.gw.go" -print -exec rm -f {} \;
	@echo "Generating Go code from .proto files..."
	@protoc --proto_path=. \
		--proto_path=pkg/vendor \
		--go_out=. \
		--go-grpc_out=. \
		--grpc-gateway_out=. \
		--validate-go_out=. \
		$(PROTO_FILES)
	@echo "Protobuf code generation complete."

# 生成openAPI文档.json
proto-openapi:
	@protoc -I. -I pkg/vendor \
	  --openapiv2_out docs \
	  --openapiv2_opt logtostderr=true,allow_merge=true,merge_file_name=openapi.json,use_go_templates=true \
	  pkg/protocol/proto/auth/*.proto \
	  pkg/protocol/proto/user/*.proto
	
# 使用 sqlc 生成数据库操作代码
sqlc-generate:
	@echo "Removing existing generated DAO files..."
	@if [ -d $(DAO_PATH) ]; then \
		echo "Cleaning $(DAO_PATH)..."; \
		rm -rf $(DAO_PATH)/*; \
	else \
		echo "$(DAO_PATH) does not exist, skipping removal."; \
	fi
	@echo "Generating Go code from SQL queries..."
	@sqlc generate
	@echo "SQLC code generation complete."

# 完整的数据库更新流程：迁移 + 生成代码
db-update: migrate-up sqlc-generate  mockdb
	@echo "Database schema updated and Go code regenerated."
dao-update: sqlc-generate mockdb
	@echo "DAO code regenerated."
# 验证数据库连接
db-check:
	@echo "Checking database connection..."
	migrate -database $(DATABASE_URL) -path db/migrations version

# 构建服务
build-auth:
	@echo "Building auth service..."
	go build -o bin/auth ./cmd/auth

build-connect:
	@echo "Building connect service..."
	go build -o bin/connect ./cmd/connect

build-device:
	@echo "Building device service..."
	go build -o bin/device ./cmd/device

build-user:
	@echo "Building user service..."
	go build -o bin/user ./cmd/user

build-all: build-auth build-connect build-device build-user
	@echo "All services built successfully."

mockdb:
	@echo "Generating mock for DAO layer..."
	mockgen -source=pkg/dao/querier.go -destination=pkg/mocks/mock_querier.go -package=mocks

e2e-start-containers:
	@echo "Starting Redis and MySQL containers (if not running)..."
	@docker start redis 2>/dev/null || docker run -d --name redis -p 6379:6379 redis:alpine
	@docker start mysql 2>/dev/null || docker run -d --name mysql -e MYSQL_ROOT_PASSWORD=azsx0123456 -p 3307:3306 mysql:8.0

# Stop containers used for e2e
e2e-stop-containers:
	@echo "Stopping Redis and MySQL containers..."
	@docker stop redis mysql || true

# Wait for services to be ready (delegates to script)
e2e-wait:
	@echo "Waiting for Redis and MySQL to be ready..."
	@./scripts/e2e_test.sh wait

# Build and start auth service for e2e
e2e-build-auth:
	@echo "Building auth service..."
	@go build -o bin/auth ./cmd/auth

# Start auth service in background and record pid
e2e-start-auth: e2e-build-auth
	@mkdir -p logs run || true
	@echo "Starting auth service in background (logs to logs/auth.log)..."
	@nohup bin/auth > logs/auth.log 2>&1 & echo $$! > run/auth.pid || true

# Stop auth service started by e2e
e2e-stop-auth:
	@if [ -f run/auth.pid ]; then \
		PID=`cat run/auth.pid`; \
		kill $$PID || true; \
		rm -f run/auth.pid; \
		echo "Stopped auth (pid $$PID)"; \
	else \
		echo "No auth pid file, skip"; \
	fi

# Run smoke tests (delegates to script)
e2e-smoke:
	@echo "Running e2e smoke tests..."
	@./scripts/e2e_test.sh smoke || true

# Full e2e flow: start containers, wait, start services, run smoke, cleanup
e2e: e2e-start-containers e2e-wait e2e-start-auth e2e-smoke e2e-stop-auth e2e-stop-containers
	@echo "E2E flow finished."

.PHONY: migrate-up migrate-down migrate-create proto sqlc-generate db-update db-check build-auth build-connect build-device build-user build-all e2e-start-containers e2e-stop-containers e2e-wait e2e-build-auth e2e-start-auth e2e-stop-auth e2e-smoke e2e