## IM-SERVER 即时通讯系统

目的主要是用于学习后端开发
选择 IM 的原因是比较经典，可参考的项目比较多，涉及的需要实现的功能比较能够体现 go 语言高并发(go 关键字、sync 包)和消息同步(channel)的特性。

## 需求部分

### 功能需求

1. 登录与注册
预期是实现登录的HTTP api，但通过gRPC Server实现的登录服务，所以需要实现网关gateway将HTTP请求转为gRPC请求，gRPC请求再转为gRPC响应

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

├── db 存储相关的目录
│ ├── migrations
│ │ └── # 存放数据库迁移脚本（用于 golang-migrate 等工具），例如 create*table.sql、升级/回滚脚本。运行 `make migrate-up` 或 `migrate` 命令时会使用这里的 SQL 文件来初始化或变更数据库结构。
│ └── queries
│ └── # 存放 sqlc 使用的 SQL 查询定义（*.sql），这些文件会被 sqlc 读取并生成类型安全的 Go 持久化层代码（`internal/db` 或 `internal/repo` 下的 \_.sql.go 文件）。

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
