- 笔记部分
- 1.完成了ER图设计思路的部分的总结
- 2.还没完成关于数据库字段设计部分
- 数据库部分
- [7.27]
- 1.使用golang-migrate包通过create_table.sql文件初始化迁移数据库 √
- 2.编写queries文件夹下的sql文件，用于生成持久化层的函数 √
- 3.使用sqlc将queries下的所有sql文件生成*.sql.go文件 √

- 代码开发部分
- 1.需要一个 websocket server，能够实现登录功能
- main -> StartWSServer -> wsHandler -> StartWSConn -> Serve -> HandleMessage -> SignIn

- 1.1 完成 func (c *Conn) Close() 功能
- 需要DeviceIntService->需要DeviceIntService的proto生成客户端存根
- RPC远程的、在logic包中的一个函数，该函数是实现了proto生成的服务端接口的结构体(DeviceIntService)的方法(ConnSignIn)
- 需要

- 1.2 完成 func (c *Conn) SignIn(packet \*connectpb.Packet)功能
- 