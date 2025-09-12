# gRPC-Gateway 重构说明

## 概述

本项目已成功使用 grpc-gateway 重构了 HTTP 到 gRPC 的转换，消除了手动 HTTP 处理器代码，提供了标准化的 HTTP 反向代理。

## 重构内容

### 1. Proto 文件更新

- `pkg/protocol/proto/auth/auth.int.proto`: 已包含 grpc-gateway 注解
- `pkg/protocol/proto/user/user.ext.proto`: 已添加 grpc-gateway 注解
- `pkg/protocol/proto/device/device.int.proto`: 保持纯 gRPC 服务，不添加 HTTP 转发

### 2. 生成代码

运行以下命令生成包含 grpc-gateway 支持的代码：

```bash
make proto
```

### 3. 新增组件

- `gateway/grpc_gateway.go`: 新的 grpc-gateway 服务器实现
- 更新了 `cmd/gateway/main.go` 使用新的服务器

## API 端点

grpc-gateway 自动生成的 HTTP 端点：

### 认证服务

- `POST /api/v1/auth/register` - 用户注册
- `POST /api/v1/auth/login` - 用户登录
- `POST /api/v1/auth/verify` - Token 验证

### 用户服务

- `POST /api/v1/user/search` - 用户搜索

## 启动方式

1. 确保 gRPC 服务正在运行：

   - Auth 服务: `localhost:50051`
   - User 服务: `localhost:50052`

2. 启动 grpc-gateway 服务器：

   ```bash
   go run ./cmd/gateway
   ```

3. 服务器将在 `localhost:8080` 启动，提供 HTTP API

## 配置

服务地址通过 `config.yaml` 配置：

```yaml
services:
  auth:
    rpc_addr: ":50051"
  user:
    rpc_addr: ":50052"
  gateway:
    port: 8080
```

## 优势

1. **自动化**: 自动生成 HTTP 处理器，无需手动编写
2. **标准化**: 统一的 HTTP 到 gRPC 转换
3. **维护性**: 减少重复代码，易于维护
4. **性能**: grpc-gateway 经过优化，性能良好
5. **错误处理**: 内置错误处理和日志记录

## 注意事项

- Device 服务保持纯 gRPC，不提供 HTTP 转发
- 需要确保 gRPC 服务在使用前启动
- grpc-gateway 使用 JSON 作为默认的 HTTP 内容类型
