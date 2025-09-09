## IM-SERVER 即时通讯系统

目的主要是用于学习后端开发
选择 IM 的原因是比较经典，可参考的项目比较多，涉及的需要实现的功能比较能够体现 go 语言高并发(go 关键字、sync 包)和消息同步(channel)的特性。

## 需求部分

### 功能需求

1. 登录与注册
   预期是实现登录的 HTTP api，但通过 gRPC Server 实现的登录服务，所以需要实现网关 gateway 将 HTTP 请求转为 gRPC 请求，gRPC 请求再转为 gRPC 响应(HTTP 请求 → API 网关路由 → 认证服务验证 → 令牌生成 → 返回)响应。
   - 测试登录与注册功能 √
     - 使用 mock DB 和 Redismock 测试 √
     - HTTP curl 版 编写 sh 脚本实现启动 redis、mysql，构建启动服务，发送 http 请求 √
     - 前端 UI 测试版
   - 编写用于转发 HTTP 请求到 gRPC 后端的 gateway √
2. 好友管理

   **2.1 好友关系基础功能** [P1 - 高优先级]

   - 发送好友请求 [P1]
     - 搜索用户（通过用户名/手机号） [P1]
     - 发送好友申请（包含验证消息） [P1]
     - 防止重复申请（同一用户 24 小时内只能申请一次） [P2]
   - 处理好友请求 [P1]
     - 查看待处理的好友申请列表 [P1]
     - 同意好友申请 [P1]
     - 拒绝好友申请 [P1]
     - 忽略好友申请（超时自动忽略） [P3]
   - 好友列表管理 [P1]
     - 获取好友列表（支持分页） [P1]
     - 删除好友（双向删除关系） [P2]
     - 查看好友详细信息 [P2]

   **2.2 好友分类功能** [P3 - 低优先级]

   - 创建分类 [P3]
     - 添加自定义分类（如：同事、朋友、家人） [P3]
     - 系统默认分类（默认分组） [P2]
   - 管理分类 [P3]
     - 修改分类名称 [P3]
     - 删除分类（好友移至默认分组） [P3]
   - 好友分组 [P3]
     - 将好友移动到指定分类 [P3]
     - 查看指定分类下的好友 [P3]

   **2.3 好友状态管理** [P2 - 中优先级]

   - 在线状态显示 [P2]
     - 实时获取好友在线状态 [P2]
   - 消息免打扰设置 [P3]
     - 设置/取消特定好友的消息免打扰 [P3]

3. 聊天功能

   **3.1 单聊基础功能** [P1 - 高优先级]

   - 消息发送 [P1]
     - 发送文本消息 [P1]
     - 消息去重（防止重复发送） [P2]
     - 消息序列号管理（保证顺序） [P1]
   - 消息接收 [P1]
     - 实时接收消息（WebSocket） [P1]
     - 离线消息拉取 [P1]
     - 消息已读状态同步 [P2]
   - 消息存储 [P1]
     - 消息持久化存储（MongoDB） [P1]
     - 消息索引优化（按会话+时间） [P2]
     - 消息删除（仅删除本地显示） [P3]

   **3.2 单聊扩展功能** [P2 - 中优先级]

   - 消息状态管理 [P2]
     - 消息发送状态（发送中/已发送/已送达/已读） [P2]
     - 消息撤回（2 分钟内可撤回） [P3]
   - 会话管理 [P2]
     - 创建会话 [P1]
     - 获取会话列表（按最后消息时间排序） [P2]
     - 清空聊天记录 [P3]
     - 置顶会话 [P3]

   **3.3 群聊基础功能** [P2 - 中优先级]

   - 群组管理 [P2]
     - 创建群聊（群主权限） [P2]
     - 邀请好友入群 [P2]
     - 退出群聊 [P2]
     - 解散群聊（仅群主） [P3]
   - 群成员管理 [P2]
     - 查看群成员列表 [P2]
     - 移除群成员（群主权限） [P3]
     - 设置管理员（群主权限） [P3]
   - 群消息 [P2]
     - 群内发送消息 [P2]
     - 群消息广播 [P2]
     - @提醒功能 [P3]

   **3.4 群聊扩展功能** [P3 - 低优先级]

   - 群设置 [P3]
     - 修改群名称 [P3]
     - 设置群公告 [P3]
     - 群二维码邀请 [P3]
   - 群权限管理 [P3]
     - 全员禁言/解除禁言 [P3]
     - 单个成员禁言 [P3]
     - 仅管理员可邀请设置 [P3]

## 近期开发计划

### Phase 1: 基础好友管理 (当前优先级)

**Week 1-2:**

- [x] 数据库设计：friend_request、friend 表结构设计与迁移、生成相应的dao层代码以及dao层代码的mock
- [x] 重构:将项目中涉及logic的重构为device
- [x] 实现用户搜索 API (GET /api/v1/user/search)
- [ ] 实现发送好友申请 API (POST /api/v1/friend/request)
- [ ] 实现查看好友申请列表 API (GET /api/v1/friend/requests)
- [ ] 实现处理好友申请 API (PUT /api/v1/friend/request/:id)

### Phase 2: 基础单聊功能 (第 3-4 周)

**Week 3-4:**

- [ ] 数据库设计：message、conversation 表结构设计
- [ ] 实现消息发送 WebSocket 协议
- [ ] 实现消息持久化存储 (MongoDB)
- [ ] 实现离线消息拉取 API
- [ ] 实现获取好友列表 API (GET /api/v1/friends)

### Phase 3: 完善单聊 (第 5-6 周)

**Week 5-6:**

- [ ] 实现消息序列号管理
- [ ] 实现会话创建与管理
- [ ] 实现消息已读状态同步
- [ ] 优化消息索引性能
- [ ] 前端 UI：好友列表 + 单聊界面

## 优先级标注思路

**P1 (高优先级) - MVP 必备功能:**

- 核心业务流程：用户搜索 → 发送申请 → 处理申请 → 建立好友关系
- 基础通讯：文本消息收发、实时通讯、消息存储
- 用户体验基础：好友列表、会话管理

**P2 (中优先级) - 体验增强:**

- 状态管理：在线状态、消息状态
- 群聊基础：小规模群聊功能
- 性能优化：分页、索引、去重

**P3 (低优先级) - 高级功能:**

- 个性化功能：分类、免打扰、置顶
- 管理功能：群权限、撤回、禁言
- 扩展功能：二维码、公告等

**制定思路:**

1. **技术依赖关系**: 先实现基础架构，再添加复杂功能
2. **用户价值**: 优先实现用户最核心的需求（加好友、聊天）
3. **开发复杂度**: 简单功能先行，复杂的状态管理和权限控制后置
4. **测试验证**: 每个 Phase 都能产出可测试的完整功能模块

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
  - 存放 sqlc 识别的 SQL 文件（SELECT/INSERT/UPDATE/DELETE 等），sqlc 会根据这些文件生成类型安全的 Go 数据访问代码（\*.sql.go）。
  - 生成命令：`sqlc generate` 或项目中的 `make sqlc-generate`。
  - 生成路径由 `sqlc.yaml` 配置控制（项目中常见输出位置：`internal/repo`、`pkg/dao` 或 `internal/db`）。
  - 编写建议：单个文件按资源划分（如 `user.sql`、`message.sql`），为复杂查询添加注释以提高可读性和可维护性。

小贴士：

- 变更迁移脚本后同时更新 `sqlc.yaml`（如需包含新文件），并执行相应的 migrate / sqlc 命令以保持本地与 CI 的一致性。
- 在 PR 中同时提交 migration、queries 与生成代码或提供生成步骤说明，便于代码审查与 CI 校验。

## 工作流

修改 migrations 中文件，用于修改数据库的表结构---使用 makefile 中的 migrate-up 指令通过 migrate 工具迁移---修改 queries 中.sql 文件用于设计 dao 层的操作数据库的函数---使用 makefile 中的 mockdb 指令,mock 出模拟数据库的 dao 层函数

### 数据库开发工作流

当需要修改数据库相关的业务逻辑时，请遵循以下标准流程。这个流程确保了数据库表结构、Go 数据访问层代码和（可选的）测试 mock 代码保持同步。

1.  **第一步：修改数据库表结构**

    - **操作**：在 `db/migrations/` 目录下创建或修改迁移文件 (`*.up.sql` 和 `*.down.sql`)。
    - **目的**：定义新的表、添加/修改列或索引。

2.  **第二步：应用数据库迁移**

    - **操作**：在终端运行 `make migrate-up` 命令。
    - **目的**：使用 `golang-migrate` 工具将你在上一步的表结构变更应用到数据库中。

3.  **第三步：定义或更新数据访问查询**

    - **操作**：在 `db/queries/` 目录下创建或修改 `.sql` 文件。
    - **目的**：使用 `sqlc` 的注释语法编写或更新 SQL 查询（`SELECT`, `INSERT`, `UPDATE` 等）。这些查询将用于生成类型安全的 Go 函数。

4.  **第四步：生成 Go 数据访问层 (DAO) 代码**
    - **操作**：在终端运行 `make sqlc-generate` 命令（或直接运行 `sqlc generate`）。
    - **目的**：`sqlc` 工具会读取 `db/queries/` 下的 SQL 文件，并自动生成或更新 `pkg/dao/` 目录下的 Go 代码，供业务逻辑层调用。

### api 服务开发工作流

- 每次更新完 api.go 后按照 api 更新的部分更新 api_test.go 的对应部分

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
