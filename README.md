
## IM-SERVER 即时通讯系统

目的主要是用于学习后端开发
选择 IM 的原因是比较经典，可参考的项目比较多，涉及的需要实现的功能比较能够体现 go 语言高并发(go 关键字、sync 包)和消息同步(channel)的特性。

## 需求部分

### 功能需求


## 笔记部分

1. 完成了 ER 图设计思路的部分总结
2. 还没完成关于数据库字段设计部分

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

### WebSocket Server

1. 需要一个 WebSocket Server，能够实现登录功能
   - `main` -> `StartWSServer` -> `wsHandler` -> `StartWSConn` -> `Serve` -> `HandleMessage` -> `SignIn`

#### 完成关闭连接的功能 ---`func (c *Conn) Close()` 功能

- 对已登录的设备从全局连接管理器中删除连接 ✅
- RPC 远程调用 DeviceIntService 中的 Offline 使得设备离线
  - 使用 `protoc` 编译 `DeviceIntService.proto` 文件，生成对应 Offline 的 Go 代码 ✅
  - 在 logic 层实现 RPC 远程调用的服务端的服务中 Offline 函数
- 关闭底层的物理连接 ✅

#### 完成登录功能 --- `func (c *Conn) SignIn(packet *connectpb.Packet)` 功能

- 需要对传入的数据进行 Unmarshal 到变量结构体中 ✅

- 需要验证登录时提交的信息的(这里是 packet.Data)正确性，涉及 DeviceIntService 的远程调用

  1.  需要通过 `.proto` 生成 RPC 远程调用 DeviceIntService 时访问的 ConnSignIn 函数代码 ✅
      - 使用 `protoc` 编译 `DeviceIntService.proto` 文件，生成对应的 ConnSignIn 的 Go 代码 ✅
      - 在 logic 层实现 RPC 远程调用的服务端的服务中的 ConnSignIn 函数 //TO DO
  2.  需要创建 RPC 客户端 (`NewDeviceIntServiceClient`) 用于调用 ✅
  3.  填充 `ConnSignIn` 的请求结构体 ✅
  4.  完成上述结构体涉及的 `config` 包中 `ServiceConfig.ConnectEndpoints` 部分 ✅

- 对远程调用 DeviceIntService 是否报错进行验证 ✅

### grpc 的客户端部分

1. 编写 config 中的 DatabaseConfig、ServiceConfig、GRPCClientConfig(各种 config)
2.

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
