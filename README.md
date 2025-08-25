## IM-SERVER 即时通讯系统

目的主要是用于学习后端开发
选择 IM 的原因是比较经典，可参考的项目比较多，涉及的需要实现的功能比较能够体现 go 语言高并发(go 关键字、sync 包)和消息同步(channel)的特性。

## 需求部分

### 功能需求

1. 登录与注册
   预期是实现登录的 HTTP api，但通过 gRPC Server 实现的登录服务，所以需要实现网关 gateway 将 HTTP 请求转为 gRPC 请求，gRPC 请求再转为 gRPC 响应
   - HTTP 请求 → API 网关路由 → 认证服务验证 → 令牌生成 → 返回响应。

## 笔记部分

1. 完成了 ER 图设计思路的部分总结
2. 还没完成关于数据库字段设计部分
3. **数据库用户认证字段扩展** [2025-08-23]：
   - 在 user 表中添加了 `hashed_password` 和 `salt` 字段用于用户密码认证
   - 创建了新的迁移文件 `000002_add_user_auth_fields.up.sql` 和对应的回滚文件
   - 更新了 `db/queries/user.sql` 文件，添加了认证相关的查询：
     - `CreateUser`：创建用户时包含密码和盐值
     - `GetUserByPhoneForAuth`：根据手机号获取用户认证信息
     - `UpdateUserPassword`：更新用户密码
   - 更新了 `sqlc.yaml` 配置以包含新的迁移文件
   - 使用 `sqlc generate` 重新生成了 Go 代码，更新了 `internal/repo` 下的文件
   - User 结构体现在包含 `HashedPassword` 和 `Salt` 字段

## 项目目录介绍

示例树状结构（便于渲染）：

```
db/
├─ migrations/      # 数据库迁移脚本（用于 golang-migrate）
│  ├─ *.up.sql
│  └─ *.down.sql
└─ queries/         # sqlc 的 SQL 查询定义（*.sql），用于生成类型安全的 Go 持久化层
   ├─ *.sql
```

- migrations/
  - 存放数据库版本化迁移脚本（up/down）。使用 golang-migrate 或 Makefile 中的命令管理数据库结构变更。
  - 常用命令：`make migrate-up`、`migrate`（根据项目 Makefile 或 CI 配置）。
  - 建议使用语义化/时间戳前缀命名（例如 `000002_add_user_auth_fields.up.sql` / `.down.sql`）。

- queries/
  - 存放 sqlc 识别的 SQL 文件（SELECT/INSERT/UPDATE/DELETE 等），sqlc 会根据这些文件生成类型安全的 Go 数据访问代码（*.sql.go）。
  - 生成命令：`sqlc generate` 或项目中的 `make sqlc-generate`。
  - 生成路径由 `sqlc.yaml` 配置控制（项目中常见输出位置：`internal/repo`、`pkg/dao` 或 `internal/db`）。
  - 编写建议：单个文件按资源划分（如 `user.sql`、`message.sql`），为复杂查询添加注释以提高可读性和可维护性。

小贴士：
- 变更迁移脚本后同时更新 `sqlc.yaml`（如需包含新文件），并执行相应的 migrate / sqlc 命令以保持本地与 CI 的一致性。
- 在 PR 中同时提交 migration、queries 与生成代码或提供生成步骤说明，便于代码审查与 CI 校验。

## 存储相关代码部分

### 数据库

1. 使用 `golang-migrate` 包通过 `create_table.sql` 文件初始化迁移数据库 ✅
2. 编写 `queries` 文件夹下的 SQL 文件，用于生成持久化层的函数 ✅
3. 使用 `sqlc` 将 `queries` 下的所有 SQL 文件生成 `*.sql.go` 文件 ✅

### Redis

1. 在 pkg 中编写了创建 redis 客户端的代码
2. device 在线状态的读写，使用 Redis 作为实时状态存储（哈希结构），供系统快速查询和更新设备在线信息。(status.go) [8.5]
3. token 验证使用 redis 管理了 user 的 device 对应的 token(auth.go)
   > auth.go 在这份文件中，Redis 使用的是哈希（Hash）数据结构：
   >
   > - **Redis 键（key）**：`fmt.Sprintf(AuthKey, userID)` → `"auth:<userID>"`（每个用户对应一个哈希）
   > - **哈希字段（field）**：`strconv.FormatUint(deviceID, 10)`（设备 ID 的字符串形式）
   > - **哈希值（value）**：`json.Marshal(device)` 生成的 JSON 字符串（二进制/字节）
   >
   > 所以可以描述为：在 Redis 中以 Hash 存储，field 为 `deviceID`，value 为序列化后的 `AuthDevice`（JSON）。

## 代码开发部分

### grpc 的客户端部分

1. 编写 config 中的 DatabaseConfig、ServiceConfig、GRPCClientConfig(各种 config)

### 前端部分

1. 使用 Vue 3 构建了前端界面。
2. 实现了登录功能：
   - 用户输入用户名和密码后，点击登录按钮触发 WebSocket 连接。
   - 前端通过 WebSocket 向后端发送登录请求。
   - 处理后端返回的响应并显示登录结果。
3. 前端代码结构：
   - `src/components/Login.vue`：实现登录界面和逻辑。
   - `src/App.vue`：集成登录组件。
4. 使用 Vite 作为开发工具，支持快速开发和热更新。

### 测试部分

1. 对创建 websocket 连接并发出 SignIn packet 的功能进行了测试 conn_test.go
2. 创建了前端，使用前端测试了下述流程
   > - `main` -> `StartWSServer` -> `wsHandler` -> `StartWSConn` -> `Serve` -> > `HandleMessage` -> `SignIn`
   >   虽然前端显示通过了，但有两个疑点 1.为什么 gRPC Server 没有开启但是使用 gRPC Client 进行 RPC 的部分没有出错？2.为什么 slog.Debug 函数没有发送结果到终端？

### 参考资料

- GlideIM - Golang 实现的高性能的分布式 IM:https://learnku.com/articles/67271

- 《从 0 到 1 搭建一个 IM 项目》 https://learnku.com/articles/74274

- fim 即时通讯微服务项目课程介绍 https://www.fengfengzhidao.com/article/JtzvhY4BEG4v2tWkjl7-

-
