# IM Server Web 前端

这是 IM Server 项目的 Web 前端界面，用于测试和演示登录功能。

## 功能特性

- ✅ WebSocket 连接到后端服务器
- ✅ 用户登录界面
- ✅ 实时连接状态显示
- ✅ 错误处理和状态反馈
- ✅ 可配置的服务器地址

## 文件结构

```
web/
├── index.html          # 主页面
├── login.js           # 登录逻辑和WebSocket处理
└── README.md          # 本文件
```

## 使用方法

### 1. 启动后端服务器

首先确保你的 IM 服务器正在运行：

```bash
# 在项目根目录下
go run cmd/connect/main.go
```

### 2. 打开前端页面

用浏览器打开 `web/index.html` 文件，或者通过 HTTP 服务器访问：

```bash
# 使用 Python 启动简单的 HTTP 服务器
cd web
python3 -m http.server 8000

# 然后访问 http://localhost:8000
```

### 3. 配置和登录

1. **服务器配置**：

   - 默认服务器地址：`ws://localhost:8080/ws`
   - 可以根据实际情况修改

2. **登录信息**：

   - 用户 ID：67890（默认）
   - 设备 ID：12345（默认）
   - Token：test-token（默认）

3. 点击"连接并登录"按钮

## 实现细节

### WebSocket 连接流程

1. **建立连接**：`websocket.DefaultDialer.Dial` → 发起 HTTP 升级请求
2. **服务器处理**：`StartWSServer` → `wsHandler` → `upgrader.Upgrade`
3. **连接建立**：`StartWSConn` → 创建 `Conn` 对象
4. **消息处理**：`Serve` → `HandleMessage` → `SignIn`

### 数据格式

前端使用简化的 Protobuf 编码来与后端通信：

```javascript
// Packet 结构
{
    command: 1,        // SIGN_IN
    requestId: timestamp,
    code: 0,
    message: "",
    data: SignInInput  // 序列化的登录数据
}

// SignInInput 结构
{
    deviceId: number,
    userId: number,
    token: string
}
```

## 注意事项

### 1. Protobuf 编码

当前实现使用了简化的 Protobuf 编码，在生产环境中应该：

- 使用 `protobuf.js` 库
- 根据 `.proto` 文件生成对应的 JavaScript 代码
- 确保与后端的编码格式完全一致

### 2. 错误处理

前端会显示以下状态：

- 🟢 **连接成功**：WebSocket 连接建立
- 🔴 **连接失败**：无法连接到服务器
- 🟡 **登录中**：正在发送登录请求
- ✅ **登录成功**：收到成功响应（code = 0）
- ❌ **登录失败**：收到错误响应或异常

### 3. 调试

打开浏览器开发者工具可以看到：

- WebSocket 连接状态
- 发送和接收的消息
- 错误日志

也可以在控制台使用 `window.imClient` 对象进行调试。

## 下一步开发

1. **完善 Protobuf 支持**：

   ```bash
   npm install protobufjs
   # 生成 JavaScript Protobuf 代码
   ```

2. **添加更多功能**：

   - 消息发送/接收
   - 好友列表
   - 群组功能

3. **UI 优化**：

   - 响应式设计
   - 更好的错误提示
   - 加载动画

4. **安全性**：
   - Token 管理
   - 输入验证
   - XSS 防护

## 测试

确保你的测试覆盖了完整的调用链：

```
前端 WebSocket 连接 → StartWSServer → wsHandler → StartWSConn → Serve → HandleMessage → SignIn
```

这样就完成了从前端到后端的完整登录流程测试。
