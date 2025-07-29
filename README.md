## 笔记部分

1. 完成了 ER 图设计思路的部分总结
2. 还没完成关于数据库字段设计部分

## 数据库部分

### [7.27]

1. 使用 `golang-migrate` 包通过 `create_table.sql` 文件初始化迁移数据库 ✅
2. 编写 `queries` 文件夹下的 SQL 文件，用于生成持久化层的函数 ✅
3. 使用 `sqlc` 将 `queries` 下的所有 SQL 文件生成 `*.sql.go` 文件 ✅

## 代码开发部分

### WebSocket Server

1. 需要一个 WebSocket Server，能够实现登录功能
   - `main` -> `StartWSServer` -> `wsHandler` -> `StartWSConn` -> `Serve` -> `HandleMessage` -> `SignIn`

#### 1.1 完成 `func (c *Conn) Close()` 功能

- 需要 `DeviceIntService`
- 需要 `DeviceIntService` 的 proto 生成客户端存根
- RPC 远程的、在 `logic` 包中的一个函数，该函数是实现了 proto 生成的服务端接口的结构体 (`DeviceIntService`) 的方法 (`ConnSignIn`)

#### 1.2 完成 `func (c *Conn) SignIn(packet *connectpb.Packet)` 功能

### grpc 的客户端部分

1. 编写 config 中关于 RPC 的 Server/Client config 结构体(还有Database、Server config结构体)
2. 
