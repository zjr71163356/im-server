# 查找项目中所有的 .proto 文件
PROTO_FILES := $(shell find . -name "*.proto")
DATABASE_URL := "mysql://root:azsx0123456@tcp(localhost:3307)/imserver?multiStatements=true"

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
	@echo "Generating Go code from .proto files..."
	@protoc --proto_path=. \
            --go_out=. \
            --go-grpc_out=. \
            $(PROTO_FILES)
	@echo "Protobuf code generation complete."

# 使用 sqlc 生成数据库操作代码
sqlc-generate:
	@echo "Generating Go code from SQL queries..."
	sqlc generate
	@echo "SQLC code generation complete."

# 完整的数据库更新流程：迁移 + 生成代码
db-update: migrate-up sqlc-generate
	@echo "Database schema updated and Go code regenerated."

# 验证数据库连接
db-check:
	@echo "Checking database connection..."
	migrate -database $(DATABASE_URL) -path db/migrations version

.PHONY: migrate-up migrate-down migrate-create proto sqlc-generate db-update db-check