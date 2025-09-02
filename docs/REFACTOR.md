# IM 服务器认证服务重构总结

## 重构概述

本次重构将用户认证功能从 `user` 服务中分离出来，创建了专门的 `auth` 认证服务，提高了系统的模块化程度和可维护性。

## 主要变更

### 1. Protocol Buffer 重构

- 将 `pkg/protocol/proto/user/user.int.proto` 重构为 `pkg/protocol/proto/auth/auth.int.proto`
- 新的认证服务定义了两个主要 RPC 方法：
  - `Auth`: 验证用户令牌
  - `Login`: 用户登录验证

### 2. 新增认证服务

- **文件位置**: `internal/auth/api.go`
- **服务端口**: 8020
- **主要功能**:
  - 用户登录验证（支持用户名/邮箱/手机号）
  - JWT 令牌生成和验证
  - 密码哈希验证（支持盐值）

### 3. 更新服务配置

- **文件**: `pkg/config/config.go`
- 添加了 `AuthEndpoints` 配置
- 添加了 `UserEndpoints` 配置
- 更新了 gRPC 客户端配置

### 4. 更新 RPC 客户端

- **文件**: `pkg/rpc/rpc.go`
- 新增 `GetAuthIntServiceClient()` 方法
- 移除了旧的用户认证相关客户端

### 5. 设备服务更新

- **文件**: `internal/device/api.go`
- 更新认证调用，使用新的 `authpb` 而不是 `userpb`

### 6. 用户服务简化

- **文件**: `internal/user/api.go`
- 移除了认证相关功能
- 专注于用户信息管理

## 服务架构

```
认证服务 (auth) - 端口 8020
├── 用户登录验证
├── JWT 令牌管理
└── 密码验证

用户服务 (user) - 端口 8030
├── 用户信息管理
└── 用户数据操作

逻辑服务 (logic) - 端口 8010
├── 设备管理
└── 业务逻辑处理

连接服务 (connect) - 端口 8000/8002
├── WebSocket 连接管理
└── 实时通信
```

## 新增文件

1. `pkg/protocol/proto/auth/auth.int.proto` - 认证服务协议定义
2. `internal/auth/api.go` - 认证服务实现
3. `cmd/auth/main.go` - 认证服务启动入口
4. `start_services.sh` - 服务启动脚本
5. `stop_services.sh` - 服务停止脚本

## 构建和运行

### 构建所有服务

```bash
make build-all
```

### 启动所有服务

```bash
./start_services.sh
```

### 停止所有服务

```bash
./stop_services.sh
```

### 重新生成 Protocol Buffer

```bash
make proto
```

## 认证流程

1. **用户登录**: 客户端调用 `auth` 服务的 `Login` 方法
2. **令牌验证**: 其他服务调用 `auth` 服务的 `Auth` 方法验证令牌
3. **密码验证**: 支持盐值的密码哈希验证
4. **多种登录方式**: 支持用户名、邮箱、手机号登录

## 技术特性

- **安全性**: 密码使用盐值哈希存储
- **可扩展性**: 独立的认证服务便于水平扩展
- **维护性**: 清晰的服务边界和职责分离
- **一致性**: 统一的认证接口和错误处理

## 注意事项

1. 确保数据库连接正常
2. Redis 服务需要启动（用于会话管理）
3. 所有服务需要按正确顺序启动
4. 确保端口 8000、8010、8020、8030 未被占用

## 下一步

1. 添加认证服务的单元测试
2. 实现令牌刷新机制
3. 添加认证日志和监控
4. 优化密码强度验证
