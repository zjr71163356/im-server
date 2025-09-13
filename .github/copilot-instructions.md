# Copilot Instructions for im-server

## 概要

im-server 是一个高性能、可扩展的分布式即时通讯（IM）服务端示例工程。项目按服务与职责模块化，覆盖网关、鉴权、用户、连接管理、设备管理、消息队列与多种存储后端等常见 IM 需求。

以下说明基于当前代码仓库结构，描述各目录/文件的用途与开发、构建、测试注意事项。

## 主要目录与文件（含义）

- cmd/ — 各服务的可执行程序入口

  - `cmd/auth/main.go`：Auth 服务入口（注册/登录/校验 token）。
  - `cmd/user/main.go`：User 服务入口（用户资料、搜索等）。
  - `cmd/gateway/main.go`：Gateway 服务入口（启动 grpc-gateway HTTP -> gRPC 转换）。
  - `cmd/connect/main.go`：Connect 服务入口（长连接管理，WebSocket/TCP）。
  - 其它 `cmd/*`：可扩展的服务入口。

- internal/ — 各服务的业务实现（不可导出，服务内用）

  - `internal/auth/`：鉴权逻辑与处理器。
  - `internal/user/`：用户逻辑（搜索、资料等）。
  - `internal/connect/`：Conn/Session、长连接、消息分发处理（`conn.go`, `hub.go`, `ws_server.go` 等）。
  - `internal/device/`：设备相关状态逻辑（仅 gRPC）。

- gateway/ — grpc-gateway 相关实现

  - `gateway/grpc_gateway.go`：grpc-gateway 注册与自定义映射、HTTP 编码/错误处理、CORS 等。

- pkg/ — 公共库与生成的协议代码

  - `pkg/protocol/proto/`：所有 .proto 源文件（按服务分目录）。
  - `pkg/protocol/pb/`：protoc 生成的 Go 代码（pb 与 gw 文件）。
  - `pkg/config/config.go`：集中配置结构与加载（YAML/JSON）。
  - `pkg/dao/`：sqlc 生成的数据库访问代码（queries、models）。
  - `pkg/redis/redis.go`：Redis 客户端封装。
  - `pkg/rpc/validation.go`：共享的 gRPC 校验拦截器（运行时会调用生成的 Validate()，将无效输入转换为 gRPC InvalidArgument）。
  - `pkg/vendor/`：项目内 vendored 的外部 proto 依赖（例如 google api proto、protoc-gen-validate 的 proto），用于离线/可重复的 proto 生成。

- db/ — 数据库迁移与 sqlc 查询

  - `db/migrations/`：数据库迁移脚本（`000001_init.up.sql`, `000001_init.down.sql`）。
  - `db/queries/`：sqlc 查询文件（user.sql, device.sql, message.sql 等）。
  - `sqlc.yaml`：sqlc 配置。

- pkg/dao/（已生成）

  - `pkg/dao/*.sql.go` 和 `models.go`：sqlc 输出的类型与查询实现。

- scripts/ — 开发、测试与启动脚本

  - `scripts/test_api.sh`：API 级别的测试/用例脚本（会调用 grpc-gateway 的 HTTP 接口，处理 login 返回的 JSON，以便提取 token/userId）。
  - `scripts/e2e_test.sh`：端到端脚本，包含启动/停止服务、端口清理与日志收集的辅助逻辑。
  - `scripts/start_services.sh`, `scripts/stop_services.sh`：本地服务快速启动/停止脚本（注意：会强杀占用端口进程以保证重启可靠性）。

- frontend/ — 前端演示（可选）

  - `frontend/src/`、`vite.config.js`：提供一个简单前端用于演示与交互（WebSocket、测试页面）。

- bin/ — 推荐用于存放编译后的服务二进制

  - 示例：`bin/auth`, `bin/gateway`, `bin/user`。

- logs/ — 运行时日志文件（各服务写入的日志，例如 `auth.log`, `gateway.log`）

- Makefile — 构建、proto、迁移与常用任务入口

  - 重要 target：`make proto`（生成 pb 与 gw 代码）、`make build`、`make migrate-up` / `migrate-down`。
  - 注意：Makefile 已配置将 `pkg/vendor`（或 `pkg/protocol/vendor`）加入 protoc 的 `--proto_path`，以支持离线生成。

- README.md / DESIGN.md / docs/ — 项目说明与设计文档

## Proto 与代码生成注意事项

- 所有 proto 源文件放在 `pkg/protocol/proto/` 对应服务子目录下。使用 `make proto` 调用 protoc 生成 Go pb 与 grpc-gateway gw 文件，输出到 `pkg/protocol/pb/`。
- 外部 proto 依赖（例如 `google/api/annotations.proto`、`google/api/field_behavior.proto`、`validate/validate.proto`）已 vendor 到 `pkg/vendor`，以保证离线构建与 CI 的稳定性。
- 若在 proto 中使用 `validate.rules`（protoc-gen-validate），需同时安装并在生成时启用 `protoc-gen-validate` 插件以产生 `Validate()` 方法，否则运行时校验不会自动触发（拦截器只在类型实现了 `Validate()` 时调用）。

## 运行时与校验

- grpc-gateway 运行在默认 HTTP 端口（项目中通常为 8080），通过 grpc-gateway 转发到相应的 gRPC 服务。
- 项目里实现了一个共享的 gRPC Unary 拦截器 `pkg/rpc/validation.go`，在各服务 `cmd/*/main.go` 中被注册用于统一前置校验（若消息类型实现 `Validate()` 则会被调用，校验失败会快速返回 InvalidArgument）。

## 存储与外部服务

- MySQL：用户/设备/群组/索引等主数据。
- Redis：在线状态、设备会话信息、短期缓存与部分离线逻辑。
- MongoDB：消息体持久化（大消息或历史消息存储）。
- 消息队列：系统设计支持 Kafka/NATS/NSQ 作为消息分发/解耦层（参见 DESIGN.md）。
- Push：APNs 与 FCM 的离线推送逻辑在设计里有说明（`DESIGN.md`）。

## 测试与 CI 建议

- 本地端到端测试：先运行依赖（MySQL/Redis 等），使用 `scripts/start_services.sh` 或直接构建并运行 `bin/*`，然后运行 `scripts/e2e_test.sh` 或 `scripts/test_api.sh`。
- proto 变更时：确保 `pkg/vendor` 中的外部 proto 已同步，且本机安装了 `protoc-gen-go`, `protoc-gen-go-grpc`, `protoc-gen-grpc-gateway` 与 `protoc-gen-validate`（如需生成 Validate 方法）。
- 数据库迁移：若修改 schema，请添加/更新 `db/migrations/*` 并在 CI 中运行 `make migrate-up` 测试迁移。

## 开发提示与约定

- 使用 Protobuf 定义服务边界与消息格式；所有对外 HTTP 映射通过 grpc-gateway 注解在 proto 中声明。
- 统一把公共逻辑放到 `pkg/`，业务实现放到 `internal/`。
- 在服务入口注册 `pkg/rpc.ValidationUnaryInterceptor()`，以在 RPC 边界统一执行消息级校验（更早地拒绝非法请求，避免 DB 错误）。
- 将外部 proto 依赖 vendor 到仓库以保证可重复构建；示例路径：`pkg/vendor/google/api/...` 和 `pkg/vendor/validate/...`。

---

如果需要把该文件进一步细化为按服务的开发流程（例如如何本地调试 auth/user/gateway 的一条典型请求链路），我可以基于当前 README/脚本添加具体步骤与命令。
