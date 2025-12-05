## IM-SERVER：面试导向版速览
gRPC+MySQL+kafka+
学习性质的项目，不需要太多细节
面向后端面试复盘，围绕“为何这么设计、权衡点、代码入口”组织。每一节都可以对应到常见问题并指向实现位置。


### 快速启动（本地一键，docker-compose）

- 依赖：MySQL、Redis、MongoDB、Kafka（`docker-compose.yml` 已内置）。
- 启动：`docker compose up -d`（默认暴露 3307/6379/27017/9092 等）。
- 数据迁移与代码生成：`make migrate-up`，`make sqlc-generate`。
- 运行服务：`make start`（若有分服务可按 `make <service>` 启动）。
- 停止：`docker compose down`。

### 架构与模块入口

- 网关：HTTP → gRPC 转发、JWT 校验（Fiber + grpc-gateway），路径 `gateway/`。
- 用户与好友：`internal/user/`、`internal/friend/`，包含搜索、申请、处理。
- 消息：`internal/message/` 负责消息入库、索引、事件发布；`internal/connect/` WebSocket 下发。
- 存储/客户端：`pkg/redis/`、`internal/repo/`（sqlc 生成）、Mongo/Kafka 客户端在 `pkg/`。
- 前端：`frontend/`（Vue3，登录 + WS 测试）。

### 核心场景与面试要点

**鉴权与安全**
- JWT 鉴权中间件，网关注入 `user_id/device_id` 到 gRPC 上下文。
- 密码存储：`user` 表包含 `hashed_password`、`salt`（迁移 `000002_add_user_auth_fields.up.sql`）。

**好友管理**
- 场景：搜索 → 发送申请（防重复） → 审批 → 建立双向关系。
- 关键：防重复申请（按申请人/被申请人/时间窗去重），列表分页，幂等处理（请求层幂等键）。
- 入口：`internal/friend/` API + DAO，缓存可放 Redis（关系读多写少）。

**单聊消息链路**
- 流程：客户端携带 `recipient_id/client_msg_id` → Gateway 校验 → Message 服务幂等校验 → 解析/创建会话 → 分配序列号 → MySQL 写索引/会话/未读 → Mongo 写消息体 → Kafka 发布 → Connect 消费并推送在线设备，离线依赖未读+历史拉取。
- 幂等键：`(sender_id, device_id, client_msg_id)`；重复请求直接返回首结果。
- 序列号：按会话或收件箱原子自增（Redis INCR + 持久化或 DB 乐观锁）。

### 数据与存储选择

- **MySQL**：关系/索引数据（好友关系、消息索引、会话、未读）。索引按 `conversation_id+seq`、`user_id+updated_at`。
- **MongoDB**：消息正文大字段与时间序列查询（`conversation_id+timestamp` 索引）。用 docker-compose 内置 `mongo`。
- **Redis**：在线状态、auth token、序列号发号器、好友/用户缓存。
- **Kafka**：事件总线 `message.deliver`，至少一次投递。

### 一致性与可靠性

- 简化：直接 DB + Kafka 发布，接受“至少一次”与偶发丢失风险，依赖幂等消费/重试。

### 高并发与性能策略

- 限流：网关/服务按 IP、用户、会话维度限流。
- WS 连接：心跳 + 空闲踢出，连接数上限；消息投递采用批量/聚合。
- 序列号发号：Redis INCR（热键保护可分片或 Lua + local cache），并将结果持久化到 DB。
- 去重/幂等：请求层幂等键 + 业务层唯一索引。

### 缓存与数据一致性

- 读多写少（好友、用户）缓存：先更新 DB，再删缓存，避免脏读；必要时读旁路回源。
- 在线状态存 Redis Hash；token 以 `auth:<user_id>` Hash 结构存储，field 为 `device_id`。
- 热点保护：本地缓存 + 随机过期。

### 安全面试点

- JWT 过期/刷新策略；接口权限校验。
- 防刷：限流 + 幂等；输入验证（`protoc-gen-validate`）。
- 数据保护：密码哈希+盐；敏感日志脱敏。

### 可观测性

- 日志：统一格式，包含 trace/request id（gRPC 拦截器注入）。
- Metrics：QPS、错误率、p95、WS 在线数、Kafka lag。
- Trace：链路覆盖 Gateway→Service→存储，定位消息延迟。

### 测试与工作流

- 单测：sqlc DAO 可 mock（mock DB/Redis），表驱动测试；Redismock 覆盖缓存逻辑。
- 集成：docker-compose 起依赖，HTTP/WS 测试脚本，验证注册/登录/好友/消息链路。
- 常用命令：`make migrate-up`、`make sqlc-generate`、`make mockdb`、`make test`、`make start/stop`。
- 前端：`cd frontend && pnpm install && pnpm dev`（登录 + WS 收发冒烟）。

### 路线图（学习向）

- 已完成：登录/注册（JWT+Redis）、好友搜索/申请/审批、基础鉴权、会话/消息表设计、前端登录+WS 验证。
- 进行中：WS 心跳/订阅链路打通（含顶号/防假在线）、消息入库+Mongo 体存储、序列号与未读累加、前端好友列表/单聊页。
- 之后可以尝试：简化版 JWT 续期（滑动过期即可）、Redis 在线态定时回写 DB、基本已读同步与离线拉取。

### 待完善清单（落地优先）

按模块列缺口与验收点，优先完成端到端收发闭环。

| 模块 | 缺口/验收标准 | 入口 |
| --- | --- | --- |
| WS 登录链路 | SignIn/心跳幂等，房间订阅+退订（含上下文注册），限流与错误码统一。 | internal/connect/conn.go, internal/connect/manager.go |
| 在线状态回写 | Redis 在线态定期回写 DB，淘汰假在线；心跳/ACK 返回新 JWT（滑动续期）。 | internal/device/status.go, internal/device/api.go, internal/connect/conn.go |
| 消息与序列 | 单聊发送 client_msg_id 幂等，Redis 序列持久化，对齐离线拉取/未读清零的基础版。 | internal/message/service.go, pkg/storage/mongo |
| 网关/HTTP | 已切换 Fiber + grpc-gateway adaptor；待补限流、统一错误码与 CORS/鉴权配置收敛。 | gateway/ |
| 前端与测试 | 前端好友列表+单聊展示，WS 收发冒烟脚本，redismock/DAO 基础单测覆盖幂等与友链校验。 | frontend/, docs/learn.md, pkg/mocks/ |

### 目录速查

- `gateway/`：HTTP→gRPC 路由、鉴权。
- `internal/friend/`：好友业务逻辑与 API。
- `internal/message/`：消息入库、索引、事件发布。
- `internal/connect/`：WS 连接管理与推送。
- `db/migrations/`：迁移脚本（包含认证字段扩展）。
- `db/queries/`、`internal/repo/`：sqlc 查询与生成代码。
- `pkg/redis/`、`pkg/kafka/`：客户端封装。
- `frontend/`：Vue3 Demo。
