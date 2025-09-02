# 查找项目中需要编译的 .proto 文件（排除vendor目录）
PROTO_FILES := $(shell find pkg/protocol/proto -name "*.proto")
DATABASE_URL := "mysql://root:azsx0123456@tcp(localhost:3307)/imserver?multiStatements=true"
DAO_PATH := "pkg/dao"


# 将 Go 的 bin 目录添加到 PATH
GOPATH := $(shell go env GOPATH)
export PATH := $(GOPATH)/bin:$(PATH)
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
		$(PROTO_FILES)
	@echo "Protobuf code generation complete."

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

build-logic:
	@echo "Building logic service..."
	go build -o bin/logic ./cmd/logic

build-user:
	@echo "Building user service..."
	go build -o bin/user ./cmd/user

build-all: build-auth build-connect build-logic build-user
	@echo "All services built successfully."

mockdb:
	@echo "Generating mock for DAO layer..."
	mockgen -source=pkg/dao/querier.go -destination=pkg/mocks/mock_querier.go -package=mocks

.PHONY: migrate-up migrate-down migrate-create proto sqlc-generate db-update db-check build-auth build-connect build-logic build-user build-all