# 查找项目中所有的 .proto 文件
PROTO_FILES := $(shell find . -name "*.proto")
migrate-up:
	migrate -database "mysql://root:azsx0123456@tcp(localhost:3307)/imserver?multiStatements=true" -path db/migrations up

migrate-down:
	migrate -database "mysql://root:azsx0123456@tcp(localhost:3307)/imserver?multiStatements=true" -path db/migrations down
proto:
	@echo "Generating Go code from .proto files..."
	@protoc --proto_path=. \
            --go_out=. \
            --go-grpc_out=. \
            $(PROTO_FILES)
	@echo "Protobuf code generation complete."
.PHONY: migrate-up migrate-down proto