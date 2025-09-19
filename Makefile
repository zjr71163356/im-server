# 使用 bash 以便在配方中使用更丰富的语法
SHELL := /bin/bash

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
	  $(PROTO_FILES)
	
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

# 构建 gateway 服务
build-gateway:
	@echo "Building gateway service..."
	go build -o bin/gateway ./cmd/gateway

# 一次性构建全部服务
build-all: build-auth build-connect build-device build-user build-gateway
	@echo "All services built successfully."

mockdb:
	@echo "Generating mock for DAO layer..."
	mockgen -source=pkg/dao/querier.go -destination=pkg/mocks/mock_querier.go -package=mocks

# ---------------------------
# 运行项目（改为纯 Makefile 管理）
# ---------------------------

# 启动依赖（MySQL/Redis/Mongo/NATS）并等待 MySQL 健康
start-deps:
	@echo "Starting dependencies (docker compose up -d)..."
	@docker compose up -d
	@echo "Waiting for MySQL container (im_mysql) to be healthy..."
	@for i in $$(seq 1 30); do \
		status=$$(docker inspect -f '{{.State.Health.Status}}' im_mysql 2>/dev/null || echo starting); \
		if [ "$$status" = "healthy" ]; then \
			echo "MySQL is healthy."; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then \
			echo "Timeout waiting for MySQL to be healthy"; \
			exit 1; \
		fi; \
		sleep 2; \
	 done

# 后台启动业务服务（auth/user/device/connect）和 gateway（不再依赖脚本，不写 PID 文件）
start-services: build-all
	@mkdir -p logs || true
	@echo "Starting services in background (logs/*.log)..."
	@nohup bin/auth     > logs/auth.log 2>&1 & echo "auth PID $$!"     || true
	@nohup bin/user     > logs/user.log 2>&1 & echo "user PID $$!"     || true
	@nohup bin/device   > logs/device.log 2>&1 & echo "device PID $$!"   || true
	@nohup bin/connect  > logs/connect.log 2>&1 & echo "connect PID $$!"  || true
	@nohup bin/gateway -config config.yaml > logs/gateway.log 2>&1 & echo "gateway PID $$!" || true
	@echo "Services started. See logs/ for output."

# 一键启动：依赖 -> 迁移 -> 构建 -> 运行
start: start-deps migrate-up start-services
	@echo "Project is up. Gateway: http://localhost:8080  Mongo Express: http://localhost:8081"

# 停止服务（按可执行路径精确匹配，不依赖脚本/PID 文件）
stop-services:
	@echo "Stopping app services..."
	@for s in auth user device connect gateway; do \
		p=$$(realpath bin/$$s 2>/dev/null || echo ""); \
		if [ -n "$$p" ] && pgrep -f "^$$p( |$$)" >/dev/null 2>&1; then \
			echo "Stopping $$s ..."; \
			pkill -f "^$$p( |$$)" || true; \
		else \
			echo "$$s not running"; \
		fi; \
	 done

stop-deps:
	@echo "Stopping dependencies (docker compose down)..."
	@docker compose down

stop: stop-services stop-deps
	@echo "Project stopped."

# 查看运行状态
status:
	@echo "Docker compose services:" && docker compose ps || true
	@echo "\nApp processes:"
	@for s in auth user device connect gateway; do \
		p=$$(realpath bin/$$s 2>/dev/null || echo ""); \
		if [ -z "$$p" ]; then echo " - $$s: binary missing"; continue; fi; \
		pgrep -fl "^$$p( |$$)" >/dev/null 2>&1 && pgrep -fl "^$$p( |$$)" | sed 's/^/ - /' || echo " - $$s: stopped"; \
	 done

# ---------------------------
# 移除脚本化 e2e 目标（保留占位说明）
# ---------------------------
# e2e 相关逻辑请改为使用专用测试工具或在 CI 中编排。

.PHONY: migrate-up migrate-down migrate-create proto sqlc-generate db-update db-check build-auth build-connect build-device build-user build-gateway build-all start-deps start-services start stop-services stop-deps stop status mockdb proto-openapi