# 数据库设计：friend_request、friend 表结构设计与迁移应该怎么做?friend_request 要存储到数据库吗？存储到 redis 会不会更好？

### `friend_request` 应该存储在哪里？数据库 vs Redis

**结论先行：强烈建议将 `friend_request` 存储在数据库（MySQL）中，而不是 Redis。**

#### 为什么存储在数据库更好？

1.  **数据持久性 (Durability)**：好友请求是重要的用户交互数据。如果 Redis 服务器因为故障、重启而没有配置好持久化（或持久化失败），所有未处理的好友请求都会丢失。这对用户体验是毁灭性的。MySQL 作为关系型数据库，其首要设计目标就是保证数据的持久性和可靠性。

2.  **事务一致性 (Transactional Consistency)**：当用户“同意”一个好友请求时，通常需要执行两个操作：

    - 更新 `friend_request` 表的状态为 `accepted`。
    - 在 `friend` 表中**双向插入**两条好友关系记录。
      将这些操作放在一个数据库事务中，可以保证它们要么全部成功，要么全部失败。如果使用 Redis + MySQL，你需要处理分布式事务的复杂性，很容易出现数据不一致（例如，请求在 Redis 中删了，但好友关系在 MySQL 中没建成功）。

3.  **数据完整性 (Data Integrity)**：通过外键约束，你可以确保 `friend_request` 中的 `requester_id` 和 `recipient_id` 必须是 `user` 表中存在的用户，防止产生无效的“幽灵”请求。Redis 无法提供这种关系约束。

4.  **查询与分析能力 (Querying & Analytics)**：未来你可能需要分析好友请求数据（例如，用户的社交活跃度、请求拒绝率等）。SQL 提供了强大的查询能力，而 Redis 在这方面非常受限。

#### Redis 的适用场景

Redis 在这个场景中可以作为**辅助缓存**，而不是主存储：

- **缓存未读请求数**：当用户登录时，可以将其未读好友请求的数量缓存到 Redis。这样，每次前端需要显示“小红点”时，直接从 Redis 读取，而不用频繁查询数据库。当请求被处理后，再更新缓存。
- **实现“24 小时内只能申请一次”**：这个功能用 Redis 的 `SET` 命令配合 `EX` (过期时间) 非常完美。当用户 A 向 B 发送请求时，你可以在 Redis 中设置一个键，如 `friend_req_lock:A_to_B`，并设置 24 小时过期。下次再申请时，先检查这个键是否存在即可。

---

### 数据库设计与迁移步骤

下面是具体的设计和操作流程，完全符合你项目的 `golang-migrate` 和 `sqlc` 工作流。

#### 第一步：设计表结构 (`friend_request` 和 `friend`)

1.  **`friend_request` 表**：用于存储好友申请的记录。

    | 列名           | 类型           | 约束/注释                                                             |
    | :------------- | :------------- | :-------------------------------------------------------------------- |
    | `id`           | `BIGINT`       | `PRIMARY KEY`, `AUTO_INCREMENT`                                       |
    | `requester_id` | `BIGINT`       | `NOT NULL`, `FOREIGN KEY (user.id)` - 申请人 ID                       |
    | `recipient_id` | `BIGINT`       | `NOT NULL`, `FOREIGN KEY (user.id)` - 接收人 ID                       |
    | `status`       | `TINYINT`      | `NOT NULL`, `DEFAULT 0` - 状态 (0: pending, 1: accepted, 2: rejected) |
    | `message`      | `VARCHAR(255)` | `DEFAULT ''` - 验证消息                                               |
    | `created_at`   | `TIMESTAMP`    | `NOT NULL`, `DEFAULT CURRENT_TIMESTAMP`                               |
    | `updated_at`   | `TIMESTAMP`    | `NOT NULL`, `DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP`   |

    **索引建议**：

    - 在 `recipient_id` 和 `status` 上创建复合索引 `(recipient_id, status)`，用于快速查询某人的待处理请求列表。
    - 在 `requester_id` 和 `recipient_id` 上创建唯一索引 `(requester_id, recipient_id)`，可以防止在数据库层面重复发送（但业务逻辑层面用 Redis 控制频率更好）。

2.  **`friend` 表**：用于存储已建立的好友关系（双向）。

    | 列名         | 类型           | 约束/注释                                           |
    | :----------- | :------------- | :-------------------------------------------------- |
    | `id`         | `BIGINT`       | `PRIMARY KEY`, `AUTO_INCREMENT`                     |
    | `user_id`    | `BIGINT`       | `NOT NULL`, `FOREIGN KEY (user.id)` - 用户 ID       |
    | `friend_id`  | `BIGINT`       | `NOT NULL`, `FOREIGN KEY (user.id)` - 好友的用户 ID |
    | `remark`     | `VARCHAR(255)` | `DEFAULT ''` - 备注名                               |
    | `created_at` | `TIMESTAMP`    | `NOT NULL`, `DEFAULT CURRENT_TIMESTAMP`             |

    **索引建议**：

    - 在 `user_id` 上创建索引，用于快速获取某人的好友列表。
    - 在 `(user_id, friend_id)` 上创建唯一索引，防止重复添加好友。

#### 第二步：创建迁移文件

根据你的项目结构，在 migrations 目录下创建新的迁移文件。假设下一个版本号是 `000003`。

1.  **创建 `000003_create_friend_tables.up.sql` 文件**：

    ```sql
    -- filepath: /home/tyrfly/im-server/db/migrations/000003_create_friend_tables.up.sql
    CREATE TABLE `friend_request` (
      `id` bigint NOT NULL AUTO_INCREMENT,
      `requester_id` bigint NOT NULL,
      `recipient_id` bigint NOT NULL,
      `status` tinyint NOT NULL DEFAULT '0' COMMENT '0: pending, 1: accepted, 2: rejected',
      `message` varchar(255) NOT NULL DEFAULT '',
      `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
      `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
      PRIMARY KEY (`id`),
      KEY `idx_recipient_status` (`recipient_id`, `status`),
      UNIQUE KEY `uk_requester_recipient` (`requester_id`, `recipient_id`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

    CREATE TABLE `friend` (
      `id` bigint NOT NULL AUTO_INCREMENT,
      `user_id` bigint NOT NULL,
      `friend_id` bigint NOT NULL,
      `remark` varchar(255) NOT NULL DEFAULT '',
      `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
      PRIMARY KEY (`id`),
      KEY `idx_user_id` (`user_id`),
      UNIQUE KEY `uk_user_friend` (`user_id`, `friend_id`)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
    ```

2.  **创建 `000003_create_friend_tables.down.sql` 文件**：

    ```sql
    -- filepath: /home/tyrfly/im-server/db/migrations/000003_create_friend_tables.down.sql
    DROP TABLE IF EXISTS `friend_request`;
    DROP TABLE IF EXISTS `friend`;
    ```

#### 第三步：应用数据库迁移

在你的项目根目录下运行 `make` 命令：

```bash
make migrate-up
```

这将会执行 `up` 脚本，在你的 MySQL 数据库中创建这两个表。

#### 第四步：定义 SQLC 查询

在 queries 目录下创建 `friend.sql` 和 `friend_request.sql` 文件，定义数据访问操作。

1.  **创建 `db/queries/friend_request.sql`**

    ```sql
    -- name: CreateFriendRequest :one
    INSERT INTO friend_request (requester_id, recipient_id, message)
    VALUES (?, ?, ?);

    -- name: GetFriendRequestByID :one
    SELECT * FROM friend_request
    WHERE id = ?;

    -- name: UpdateFriendRequestStatus :exec
    UPDATE friend_request
    SET status = ?
    WHERE id = ?;

    -- name: ListPendingFriendRequests :many
    SELECT * FROM friend_request
    WHERE recipient_id = ? AND status = 0
    ORDER BY created_at DESC;
    ```

2.  **创建 friend.sql**

    ```sql
    -- name: AddFriend :exec
    INSERT INTO friend (user_id, friend_id)
    VALUES (?, ?);

    -- name: ListFriends :many
    SELECT * FROM friend
    WHERE user_id = ?;

    -- name: DeleteFriend :exec
    DELETE FROM friend
    WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?);
    ```

#### 第五步：生成 Go 代码

最后，运行 `sqlc` 生成 Go 代码：

```bash
make sqlc-generate
```

现在，你就可以在你的 Go 业务逻辑代码中，通过 dao 调用这些新生成的、类型安全的方法来操作好友和好友请求了。这个流程完全符合你项目的既定规范。

# 安装: `protoc-gen-validate`

好的，我们来总结一下 `protoc-gen-validate` 的整个安装、调试过程，以及最终的解决方案。

### 目标

我们的目标是在 gRPC 服务层实现自动的请求参数校验。具体来说，是利用 `protoc-gen-validate` 插件，根据 `.proto` 文件中定义的校验规则（如字段长度、范围等），自动生成 Go 代码中的 `Validate()` 方法。这样，通过 gRPC 拦截器调用此方法，就能在业务逻辑执行前拒绝无效请求，避免像“超长用户名”那样导致数据库写入失败而产生 500 错误。

---

### 安装与调试过程中的问题

我们在安装和配置过程中遇到了一系列环环相扣的问题：

1.  **问题一：`go install` 路径错误**

    - **尝试的命令**：`go install github.com/envoyproxy/protoc-gen-validate/protoc-gen-validate@latest`
    - **遇到的错误**：`...module ... does not contain package...`
    - **原因**：我们使用了旧的包安装路径。`protoc-gen-validate` 项目的目录结构已经更新，其 Go 插件的 `main` 包不再位于项目根目录，而是移到了 `cmd/protoc-gen-validate-go` 子目录中。

2.  **问题二：`protoc` 插件名称不匹配**

    - **解决问题一后**：我们使用正确的路径 `go install .../cmd/protoc-gen-validate-go@latest` 成功安装了插件，但这在 `GOPATH/bin` 中生成的是名为 `protoc-gen-validate-go` 的可执行文件。
    - **遇到的错误**：当运行 `make proto` 时，`protoc` 报错 `protoc-gen-validate: program not found...`。
    - **原因**：`protoc` 在处理 `--validate_out` 标志时，默认会去 `PATH` 中寻找名为 `protoc-gen-validate` 的可执行文件，而我们安装的却是 `protoc-gen-validate-go`，两者名称不匹配。

3.  **问题三：创建符号链接时文件已存在**

    - **尝试的命令**：为了解决名称不匹配问题，我们尝试创建一个符号链接 `ln -s .../protoc-gen-validate-go .../protoc-gen-validate`。
    - **遇到的错误**：`ln: ... File exists`。
    - **原因**：之前可能存在一个不正确或不完整的安装残留，导致 `protoc-gen-validate` 这个文件或链接已经存在。

4.  **问题四：`protoc` 插件参数格式错误**
    - **解决问题三后**：我们使用 `ln -sf` 强制创建了符号链接，并修改 Makefile 添加了 `--validate_out=lang=go:.` 参数。
    - **遇到的错误**：`[validator]unknown 'lang' parameter`。
    - **原因**：这是最关键的一步。这个错误表明我们使用的插件是**新版本**的，而 Makefile 中的 `--validate_out=lang=go:.` 是**旧版本**的参数格式。新版插件不再接受 `lang=go` 这种形式的参数。

---

### 最终的解决方案

综合以上所有问题，最终形成了一套清晰、可行的解决方案：

1.  **正确安装插件**：
    使用 cmd 下的精确路径来安装 Go 版本的插件。

    ```bash
    go install github.com/envoyproxy/protoc-gen-validate/cmd/protoc-gen-validate-go@latest
    ```

2.  **更新 Makefile 以使用新版插件参数**：
    修改 Makefile 中的 `proto` 任务，将旧的 `--validate_out=lang=go:.` 替换为新版的 `--validate-go_out=.`。这个标志会直接调用 `protoc-gen-validate-go` 插件。

    ```makefile
    # Makefile
    # ...
    proto:
        # ...
        @protoc --proto_path=. \
            --proto_path=pkg/vendor \
            --go_out=. \
            --go-grpc_out=. \
            --grpc-gateway_out=. \
            --validate-go_out=. \
            $(PROTO_FILES)
    # ...
    ```

    - **注意**：采用这种方式后，就不再需要创建 `protoc-gen-validate` 的符号链接了，因为 `protoc` 会根据 `--validate-go_out` 自动寻找 `protoc-gen-validate-go`。

3.  **重新生成代码**：
    执行 `make proto`。

    ```bash
    make proto
    ```

完成以上步骤后，`protoc` 成功调用了新版插件，并在 authpb 目录下生成了包含 `Validate()` 方法的 auth.int.pb.go 和 `auth.int.pb.validate.go` 文件，从而彻底解决了问题。现在，gRPC 拦截器可以正常工作，对超长用户名等非法请求返回 `400 Bad Request`。

# proto 设计|安全: SendFriendRequestRequest 为什么不设计成 gRPC 请求中存放请求方 userId？

您提出了一个非常好的设计问题。让我分析一下这两种设计方案的优缺点：

## 方案对比

### 方案 1：从 gRPC 请求中获取 userId（您建议的方案）

```protobuf
message SendFriendRequestRequest {
    uint64 user_id = 1;          // 请求方用户ID
    uint64 recipient_id = 2;     // 接收方用户ID
    string message = 3;          // 申请消息
}
```

### 方案 2：从 Context 中获取 userId（当前实现）

```protobuf
message SendFriendRequestRequest {
    uint64 recipient_id = 1;     // 接收方用户ID
    string message = 2;          // 申请消息
}
```

## 详细分析

### 方案 1 的优点：

1. **简单直接**：客户端直接在请求中传递 userId，服务端直接使用
2. **无需复杂的认证流程**：避免了 middleware/拦截器的复杂性
3. **便于测试**：测试时可以直接指定任意 userId
4. **跨服务调用友好**：服务间调用时可以明确指定代理的用户

### 方案 1 的缺点：

1. **安全风险巨大**：客户端可以伪造任意用户身份
2. **违反零信任原则**：完全信任客户端传递的身份信息
3. **无法防止恶意攻击**：攻击者可以冒充任何用户发送好友申请
4. **不符合现代安全最佳实践**

### 方案 2 的优点：

1. **安全性高**：通过认证 token 验证用户身份，无法伪造
2. **符合零信任架构**：不信任客户端传递的身份信息
3. **防止权限越界**：确保用户只能操作自己的资源
4. **符合行业标准**：JWT、OAuth2 等现代认证方案的标准做法

### 方案 2 的缺点：

1. **实现复杂**：需要认证 middleware/拦截器
2. **服务间调用复杂**：需要传递认证上下文
3. **测试相对复杂**：需要模拟认证流程

## 推荐方案与理由

**强烈推荐方案 2（从 Context 获取 userId）**，理由如下：

### 1. 安全性是最高优先级

```go
// 方案1：极度危险
// 客户端可以发送：{"user_id": 12345, "recipient_id": 67890}
// 冒充用户12345向用户67890发送好友申请

// 方案2：安全
// 服务端通过 token 验证确认请求确实来自合法用户
```

### 2. 现实场景的安全需求

在真实的生产环境中，IM 系统面临的安全威胁包括：

- 身份伪造攻击
- 批量垃圾好友申请
- 恶意用户冒充他人

### 3. 行业最佳实践

主流 IM 系统（微信、QQ、Telegram 等）都采用类似方案 2 的认证机制。

## 混合方案：兼顾安全与灵活性

可以设计一个混合方案，在内部服务调用时支持指定 userId：

```protobuf
message SendFriendRequestRequest {
    uint64 recipient_id = 1;
    string message = 2;
    // 仅用于内部服务调用，客户端调用时忽略
    uint64 internal_user_id = 3 [(google.api.field_behavior) = OPTIONAL];
}
```

```go
func (s *FriendExtService) SendFriendRequest(ctx context.Context, req *friendpb.SendFriendRequestRequest) (*friendpb.SendFriendRequestResponse, error) {
    var userID uint64

    // 优先从认证上下文获取（客户端调用）
    if contextUserID, ok := ctx.Value("user_id").(uint64); ok {
        userID = contextUserID
    } else if req.InternalUserId > 0 {
        // 内部服务调用时使用（需要额外的内部认证机制）
        if !isInternalCall(ctx) {
            return nil, status.Error(codes.PermissionDenied, "internal_user_id only for internal calls")
        }
        userID = req.InternalUserId
    } else {
        return nil, status.Error(codes.Unauthenticated, "user not authenticated")
    }

    // 业务逻辑...
}
```

## 总结

虽然方案 1 在开发和测试阶段更简单，但从安全性、可维护性和行业标准来看，**方案 2 是正确的选择**。安全性问题一旦发生，后果往往是灾难性的，而认证机制的复杂性是一次性的投入，带来的是长期的安全保障。

建议继续推进当前的 Context 方案，并逐步完善认证 middleware 的实现。

# 授权鉴权机制说明

基于对 im-server 项目代码的分析，我来总结当前的签名授权机制实现方式，并对比其他方案的优劣。

## 当前项目的授权机制实现

### 1. **JWT 基础实现**

```go
// pkg/jwt/jwt.go - 当前实现
type Claims struct {
    UID uint64 `json:"uid"`
    DID uint64 `json:"did"`
    jwt.RegisteredClaims
}

// 生成JWT Token
func GenerateJWT(uid, did uint64, ttl time.Duration, secret []byte, iss, aud string) (string, error)

// 解析JWT Token
func ParseJWT(tokenStr string, secret []byte, iss, aud string) (uint64, uint64, error)
```

### 2. **gRPC 拦截器验证**

```go
// pkg/rpc/auth_interceptor.go - 统一验证入口
func JWTAuthUnaryInterceptor() grpc.UnaryServerInterceptor {
    // 1. 从 metadata 提取 Authorization Bearer token
    // 2. 调用 jwt.ParseJWT 验证签名和过期时间
    // 3. 将 user_id/device_id 注入到 context
    // 4. 防止 alg 攻击，强制使用 HMAC 签名方法
}
```

### 3. **Auth 服务集中签发**

```go
// internal/auth/api.go - Token 签发中心
func (s *AuthIntService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
    // 验证用户凭据后，生成JWT返回给客户端
    token, err := jwt.GenerateJWT(user.ID, req.DeviceId, ttl, secret, iss, aud)
}
```

### 4. **Gateway 层透传**

```go
// gateway/grpc_gateway.go - HTTP到gRPC的认证透传
// Authorization header 自动映射到 gRPC metadata
```

## 优劣对比分析

### 当前方案：JWT + HMAC (HS256)

#### ✅ **优势**

- **高性能**: 对称加密验证速度快，适合高并发 IM 场景
- **无状态**: 各微服务独立验证，无需依赖 auth 服务，支持水平扩展
- **简单部署**: 只需共享一个 secret，无需管理公私钥对
- **分布式友好**: 完美契合项目的微服务架构

#### ❌ **劣势**

- **无法撤销**: Token 签发后无法主动失效（用户注销/踢下线困难）
- **密钥泄露风险**: 所有服务共享同一个 secret
- **缺乏细粒度控制**: 无法实现会话管理、设备管理等复杂场景

### 对比方案

#### 1. **JWT + RSA (RS256)**

```go
// 公私钥分离的非对称签名
type RSATokenManager struct {
    privateKey *rsa.PrivateKey // auth 服务独有
    publicKey  *rsa.PublicKey  // 其他服务共享
}
```

**优势**:

- 密钥分离：auth 服务私钥签发，其他服务公钥验证
- 更高安全性：即使公钥泄露也无法伪造 token

**劣势**:

- 性能开销：RSA 验证比 HMAC 慢 10-100 倍
- 部署复杂：需要 JWKS 管理公钥轮换
- 对本项目过度设计

#### 2. **Session + Redis**

```go
// 传统会话管理
type SessionManager struct {
    rdb redis.Cmdable
}

func (sm *SessionManager) ValidateSession(sessionID string) (*UserSession, error) {
    // 每次请求查询 Redis
}
```

**优势**:

- 完全可控：可随时撤销任何会话
- 精确统计：准确的在线用户、设备管理
- 灵活权限：支持细粒度权限控制

**劣势**:

- 性能瓶颈：每次请求都要查询 Redis
- 单点故障：Redis 故障影响所有验证
- 状态依赖：不适合分布式扩展

#### 3. **OAuth 2.0 + OpenID Connect**

```go
// 标准化的授权框架
type OAuthProvider struct {
    authServer   string
    clientID     string
    clientSecret string
}
```

**优势**:

- 标准化：业界成熟标准，易于集成第三方
- 功能完整：支持授权码、刷新 token 等多种流程
- 生态丰富：大量现成的库和工具

**劣势**:

- 复杂度高：对 IM 场景过度复杂
- 性能开销：多次网络调用验证
- 依赖外部：需要额外的 OAuth 服务器

## 推荐的优化方案

基于项目特点，我推荐 **JWT + Redis 混合方案**：

### 增强实现

```go
// pkg/jwt/enhanced_jwt.go - 增强版JWT
type EnhancedClaims struct {
    UID uint64 `json:"uid"`
    DID uint64 `json:"did"`
    JTI string `json:"jti"` // JWT ID，用于黑名单
    jwt.RegisteredClaims
}

// pkg/redis/token_manager.go - Token管理器
type TokenManager struct {
    rdb redis.Cmdable
}

// 黑名单机制
func (tm *TokenManager) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
    return tm.rdb.Set(ctx, fmt.Sprintf("blacklist:%s", jti), "1", ttl).Err()
}

func (tm *TokenManager) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
    result := tm.rdb.Exists(ctx, fmt.Sprintf("blacklist:%s", jti))
    return result.Val() > 0, result.Err()
}

// 设备会话管理
func (tm *TokenManager) StoreActiveToken(ctx context.Context, userID, deviceID uint64, jti string) error {
    key := fmt.Sprintf("user:token:%d:%d", userID, deviceID)
    return tm.rdb.Set(ctx, key, jti, time.Hour*24).Err()
}
```

### 增强的拦截器

```go
// pkg/rpc/enhanced_auth_interceptor.go
func EnhancedJWTAuthInterceptor(tokenManager *redis.TokenManager) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        // 1. 解析JWT获取 uid/did/jti
        uid, did, jti, err := jwt.ParseEnhancedJWT(token, secret, iss, aud)

        // 2. 检查黑名单（只有需要时才查询Redis）
        if blacklisted, err := tokenManager.IsBlacklisted(ctx, jti); blacklisted {
            return nil, status.Error(codes.Unauthenticated, "token revoked")
        }

        // 3. 注入用户信息
        ctx = context.WithValue(ctx, "user_id", uid)
        ctx = context.WithValue(ctx, "device_id", did)

        return handler(ctx, req)
    }
}
```

### 注销功能实现

```go
// internal/auth/api.go - 增加注销接口
func (s *AuthIntService) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
    jti := ctx.Value("token_id").(string)

    // 将当前token加入黑名单
    if err := s.tokenManager.BlacklistToken(ctx, jti, time.Hour*24*7); err != nil {
        return nil, status.Error(codes.Internal, "logout failed")
    }

    return &authpb.LogoutResponse{Message: "注销成功"}, nil
}

// 踢下线功能
func (s *AuthIntService) KickDevice(ctx context.Context, req *authpb.KickDeviceRequest) (*authpb.KickDeviceResponse, error) {
    // 获取目标设备的活跃token并加入黑名单
    activeJTI, err := s.tokenManager.GetActiveToken(ctx, req.UserId, req.DeviceId)
    if err == nil && activeJTI != "" {
        s.tokenManager.BlacklistToken(ctx, activeJTI, time.Hour*24*7)
    }

    return &authpb.KickDeviceResponse{Message: "设备已下线"}, nil
}
```

## 推荐方案的优势

### 1. **性能最优**

- 99% 的请求只做 JWT 本地验证（无网络 I/O）
- 仅在注销/踢下线时查询 Redis 黑名单
- 完美适合高并发 IM 场景

### 2. **功能完整**

- ✅ 支持用户主动注销
- ✅ 支持管理员踢下线
- ✅ 支持多设备登录管理
- ✅ 支持异地登录检测

### 3. **运维友好**

- 黑名单自动过期，无需手动清理
- 可监控异常 token 使用
- 支持灰度发布（新旧 token 并存）

### 4. **渐进式升级**

- 基于现有 JWT 实现，改动最小
- 向后兼容，可平滑升级
- 不影响现有的微服务架构

这种混合方案既保持了 JWT 的无状态优势，又通过 Redis 提供了有状态的精确控制能力，是当前项目最适合的升级路径。

# 无状态（Stateless）与有状态（Stateful）服务的核心区别是？

核心区别在于**服务本身是否保存客户端的会话状态（上下文信息）**。

---

### 1. 无状态服务 (Stateless)

- **不保存状态**：服务处理完请求后，不会保留任何关于该次交互的记忆。每个请求都必须包含处理它所需的全部信息。
- **易于扩展**：任何一个服务实例都可以处理任何请求，因此可以简单地通过增加服务器数量来水平扩展。
- **高可用**：单个实例宕机不影响用户，请求可以无缝切换到其他健康实例。
- **状态外部化**：状态被存储在外部共享存储中（如数据库、Redis）或由客户端自己携带（如 JWT Token）。
- **在您的项目中**：`auth`, `user`, `friend`, `message` 服务都是无状态的。它们处理请求所需的数据要么来自请求本身，要么来自外部的 MySQL/Redis。

### 2. 有状态服务 (Stateful)

- **保存状态**：服务会在自身内存中为客户端维持一个会话状态。后续的请求会依赖于这个已存在于服务器上的状态。服务器需要“记住”客户端是谁，以及它当前处于什么状态。

- **扩展复杂**：通常需要“会话粘滞”（Sticky Session），确保来自同一客户端的请求总是被路由到保存其状态的同一个实例上。
- **故障敏感**：单个实例宕机将导致其维护的所有会话状态丢失，影响用户体验。
- **状态内部化**：状态主要存在于服务实例的内存中。
- **在您的项目中**：`connect` 服务是典型的有状态服务，因为它维护着与客户端之间的 WebSocket **长连接**，这个连接本身就是一种状态。

好的，我来解释无状态（Stateless）与有状态（Stateful）服务的定义，并结合您的 `im-server` 项目进行说明。

这是分布式系统设计中的一个核心概念，直接关系到服务的可扩展性、可靠性和复杂性。

---

### 1. 无状态服务 (Stateless Service)

**定义**：
无状态服务**不保存**任何关于客户端先前交互的上下文信息（状态）。每一次请求对于服务器来说都是全新的、独立的，它包含了处理该请求所需的所有信息。服务器处理完请求后，不会“记住”任何关于这次交互的事情。

**通俗比喻**：
就像一个**自动售货机**。你投入硬币，按下按钮，它吐出饮料。它不关心你是谁，也不记得你上次买了什么。每次购买都是一次独立的交易。

**关键特征**：

- **无会话状态**：服务器自身内存中不存储任何客户端的会话数据。
- **极易水平扩展**：因为任何一台服务器实例都可以处理任何请求，所以可以简单地在负载均衡器后面增加更多的服务实例来分担压力。
- **高可用性**：如果一个实例宕机，负载均衡器可以无缝地将请求转发到另一个健康的实例，不会丢失任何会话信息，因为信息根本就不在服务器上。
- **状态外部化**：服务的状态被存储在外部系统中，例如：
  - **数据库/缓存** (MySQL, Redis)：用户的个人资料、好友关系等。
  - **客户端**：通过 Token (如 JWT) 携带用户身份信息。

**在您的 `im-server` 项目中**：
`auth`, `user`, `friend`, `message` 这些通过 gRPC 提供核心业务逻辑的服务，都应该是**无状态**的。

- **为什么？** 当 gateway 收到一个“添加好友”的 HTTP 请求时，它会转发给 `friend` 服务。这个 gRPC 请求本身包含了发起者 `UserID` (来自 JWT) 和目标好友的 `ID`。`friend` 服务处理这个请求，在 MySQL 中创建一条好友申请记录，然后就完成了。它不需要记住“这个用户正在申请好友”。下一次该用户查看好友列表的请求，可以由 `friend` 服务的任何一个实例来处理。

---

### 2. 有状态服务 (Stateful)

**定义**：
有状态服务**会保存**并维护客户端的交互状态（上下文）。后续的请求会依赖于服务器上已存储的这个状态。服务器需要“记住”客户端是谁，以及它当前处于什么状态。

**通俗比喻**：
就像一通**电话**。一旦连接建立，双方就可以连续对话。对话的上下文是持续的，你下一句说什么和上一句是相关的。这个“连接”本身就是一种状态。

**关键特征**：

- **维持会话状态**：服务器在内存中为每个客户端维持一个会话（Session）。
- **扩展复杂**：简单地增加实例是不够的。需要确保来自同一客户端的后续请求被路由到**同一个**保存其状态的服务器实例上（这称为“会话粘滞”或“Sticky Sessions”）。
- **故障恢复困难**：如果一个实例宕机，它内存中保存的所有客户端状态都会丢失。客户端需要重新建立连接和状态，体验较差。
- **状态内部化**：状态主要存储在服务实例的内存中。

**在您的 `im-server` 项目中**：
`connect` 服务是典型的**有状态**服务。

- **为什么？** `connect` 服务的核心职责是维护与客户端之间的 WebSocket 长连接。
  - 当一个客户端通过 WebSocket 连接上来，`connect` 服务的一个实例会为它创建一个 `Conn` 对象，并记录其 `UserID`、`DeviceID` 等信息。
  - 这个**TCP/WebSocket 连接本身就是一种状态**。
  - 当 `message` 服务需要向这个用户推送消息时，它必须通过某种方式找到维持着这个用户连接的那个**特定的 `connect` 服务实例**，并将消息发给它，再由它通过 WebSocket 连接推送给客户端。

---

### 总结对比

| 特性                 | 无状态服务 (Stateless)              | 有状态服务 (Stateful)            |
| :------------------- | :---------------------------------- | :------------------------------- |
| **状态存储**         | 外部 (数据库、Redis、客户端 Token)  | 内部 (服务实例内存)              |
| **请求处理**         | 每个请求都独立、自包含              | 后续请求依赖于先前状态           |
| **水平扩展**         | **非常容易**，直接增加实例          | **复杂**，需要会话粘滞或状态同步 |
| **可靠性**           | **高**，实例故障无影响              | **低**，实例故障导致状态丢失     |
| **`im-server` 示例** | `auth`, `user`, `friend`, `message` | `connect`                        |

### 对您项目设计的意义

您的项目设计体现了将**无状态计算**与**有状态连接**分离的最佳实践：

1.  **核心业务逻辑（无状态）**：将注册、登录、好友管理、消息处理等业务逻辑做成无状态的 gRPC 服务，使得这部分最核心、最复杂的逻辑可以被轻松地扩展和维护。
2.  **连接管理（有状态）**：将管理海量长连接的复杂性隔离在 `connect` 服务中。虽然 `connect` 服务是有状态的，但可以通过将“用户在哪台 `connect` 节点上”这个路由信息存储在 Redis 中，来降低其状态性，从而实现 `connect` 层的水平扩展。

# 建立了连接为什么还要通过 HTTP 接口调用 SendMessage，而不是 websocket

为什么已建立 WebSocket 还用 HTTP 调 SendMessage：

- 职责分离更清晰
  - WebSocket：只做“下行推送/广播”（在线投递、心跳、会话保持）。
  - HTTP/gRPC：做“上行写入”（鉴权、参数校验、幂等、事务、落库、出箱/投递）。
- 可观测性与治理更好
  - HTTP 有完善的网关/限流/熔断/重试/指标体系，便于审计与排错。
  - 超时、重试、幂等键（client_msg_id）在 HTTP 语义更自然。
- 稳定性更高
  - 移动网络场景 WS 易断，HTTP 短连接重试成本低；断线也能补发。
- 架构解耦
  - Connect 专注连接与下行广播；Message 服务专注写路径与出站事件。避免把业务写路径塞进长连层。

当然，也可以支持“WS 上行发送”：

- 方案：在 WS 协议里加一个 SendMessage 命令。Connect 收到后转调 Message RPC，成功后通过 WS 回 ACK；事件照常进 Kafka，再由 Connect 的消费者推送给接收方。
- 代价：Connect 需承担鉴权、限流、转发、重试、幂等等写路径职责，复杂度上升。

当前项目默认：HTTP/网关发消息，上行更稳；WebSocket 只做下行推送。若你更想贴近 IM 客户端“纯长连”的体验，我可以按你现有的 `connectpb.Packet` 增加上行命令并落地实现。

# 对本项目的期望是作为类似 qq/微信那样的后端，希望测试发送信息时更接近真实场景，应该使用什么方式测试发送信息的函数

我是 GitHub Copilot。

建议用“端到端、黑盒”的方式，尽量还原 QQ/微信的真实链路来测。

推荐测试方式（按真实度从高到低）:

- 真机/浏览器 WebSocket 连线
  - 启动 Connect 服务（WS: `:8082`），分别用用户 A、用户 B 登录后拿到各自 token。
  - 在两台“设备”（两份 wscat/websocat 或一个前端页 + 一个 wscat）用 `ws://localhost:8082/ws?token=...` 建立连接。
  - 通过网关的 HTTP 接口调用 SendMessage，把 A→B 的消息发出（携带 recipient_id、client_msg_id、content）。
  - 期望：B 的所有在线设备都能收到 Kafka 推来的 deliver 事件，Connect 经 `DeliverToUser` 广播，客户端弹出一条 JSON 消息负载。
- Kafka 侧验证
  - 在 Kafka 容器里用 console-consumer 订阅 `im.message.deliver`，确认事件确实入了队列。
- Outbox 兜底链路
  - 停掉 Kafka，再发消息：MySQL 的 outbox_events 应出现一条待发送记录；启动 Outbox 分发器后应补发到 Kafka。
- 幂等与多设备
  - 用相同 client_msg_id 重复调用 SendMessage，应只产生一次有效入库/未读+1；B 的多设备应都能收到同一条 deliver。

需要启动的进程:

- Connect（已内置 Kafka 消费并调用 `DeliverToUser` 推送）
- Message（提供 SendMessage，写 Mongo、Outbox、MySQL，并尝试直发 Kafka）
- Outbox 分发器（cmd/outbox/main.go，会扫描 outbox_events 并发布到 Kafka）
- Indexer（可选，cmd/indexer/main.go，用于从 deliver 事件回填 MySQL 索引/未读，保证最终一致性）

你的第二个问题：本项目是否已实现“扫描 outbox 表然后重新发送”？

- 已实现。位置在 main.go：
  - 周期性查询待发送的 outbox_events
  - 发布到 `${topic_prefix}.message.deliver`（默认 `im.message.deliver`）
  - 成功后标记已发送，保证消息最终至少一次投递。

若需要，我可以提供一套具体的测试清单（登录拿 token → 建立两端 WS → 发消息 → 看 Kafka/Connect 日志与客户端收包），或编写一个简单的前端页专门用来发消息与收消息，便于你一键演示整链路。

## Kafka Writer.WriteMessages 方法说明

```go
func (w *kafka.Writer) WriteMessages(ctx context.Context, msgs ...kafka.Message) error
```

- 功能：向写入器配置的 Kafka 主题批量写入消息。
- 同步/异步写入：
  - 若写入器未配置为异步模式，本方法会阻塞直到所有消息写入完成或达到最大重试次数。
  - 同步模式下，当批量大小大于 1 时，会等待直到：
    1. 收集到足够的消息组成完整批量，或
    2. 达到批量超时（BatchTimeout）。
  - 批量和超时是按分区评估的，不同分区可能有不同触发条件，调度器（Balancer）也会影响写入时机。
- 返回值：
  - 写入失败时返回 error，类型可能是 kafka.WriteError，可用于判断每条消息的状态。
  - 上层可通过 ctx 取消操作，取消时可能已部分写入，需整体重写。

## 本项目涉及的 Kafka 关键知识点

1. **Topic 前缀与主题命名**：
   - 在 `config.yaml` 中通过 `broker.topic_prefix` 指定前缀（默认为 `im`），最终消费/生产的主题形如 `im.message.deliver`。
2. **生产者封装**：
   - `broker.NewKafkaProducer` 创建 `KafkaProducer`，内部使用 `kafka.Writer`，并设置 `LeastBytes` 负载均衡和 `BatchTimeout`。
   - 调用 `Publish(ctx, topic, key, payload)` 写入消息，`key` 用于分区和局部顺序保证。
3. **消费者封装**：
   - `broker.NewKafkaConsumer` 创建 `KafkaConsumer`，使用 `GroupID` 和 `GroupTopics` 实现消费者组。
   - `Start(ctx, handler)` 循环调用 `reader.ReadMessage`，并将消息交给业务处理回调。
4. **Outbox 模式**：
   - 写入业务数据库后将事件存入 `outbox_events` 表，定时扫描并调用 `Publish` 可靠投递。
   - 成功后标记已发送，保证消息最终至少一次投递。
5. **实时投递与回填**：
   - Connect 服务直接消费 `im.message.deliver` 并通过 WebSocket 推送给在线设备。
   - Indexer 服务消费同一主题，将事件回写 MySQL 索引和未读数，实现最终一致性。
6. **幂等和错误处理**：
   - 生产者可重试，写失败时保留在 Outbox 继续重试。
   - 消费者侧需捕获 JSON 解析错误、业务错误后跳过或重试。
7. **监控与扩展**：
   - 可配合 Kafka 控制台工具（console-consumer、kafka-topics）观察消息流。
   - 在生产者/消费者封装中可添加指标、结构化日志和重试策略，满足高可用需求。

## Kafka 分区数和副本数详解

### 分区数 (Partitions)

**定义**: 分区是 Kafka 主题的水平切分单位，一个主题可以分为多个分区，每个分区是一个有序的消息序列。

#### 分区的作用

1. **并行处理**：

   ```bash
   # 一个主题有 3 个分区
   im.message.deliver-0  [msg1, msg2, msg3, ...]
   im.message.deliver-1  [msg4, msg5, msg6, ...]
   im.message.deliver-2  [msg7, msg8, msg9, ...]
   ```

   - 多个消费者可以同时消费不同分区，提高吞吐量
   - 生产者可以并行写入多个分区

2. **水平扩展**：

   - 分区数决定了消费者组内最大并发消费者数量
   - 分区数 = 3，最多只能有 3 个消费者同时工作

3. **负载均衡**：
   ```go
   // 在本项目中，使用 conversation_id 作为 key
   producer.Publish(ctx, topic, []byte(convID), payload)
   ```
   - 相同 key 的消息会路由到同一分区，保证顺序
   - 不同 key 的消息分散到不同分区，实现负载均衡

#### 分区数的选择策略

**选择原则**：

- **IM 场景建议**：3-6 个分区起步
- **计算公式**：分区数 ≥ 预期的消费者实例数
- **考虑因素**：
  - 消息量：高并发场景需要更多分区
  - 顺序性：需要严格顺序的场景分区不宜过多
  - 资源开销：每个分区占用文件句柄和内存

### 副本数 (Replicas)

**定义**: 副本是分区数据的备份，用于保证数据的可靠性和高可用性。

#### 副本的作用

1. **数据备份**：

   ```bash
   # 副本数 = 3 的情况
   Broker-1: [分区0-主副本] [分区1-从副本] [分区2-从副本]
   Broker-2: [分区0-从副本] [分区1-主副本] [分区2-从副本]
   Broker-3: [分区0-从副本] [分区1-从副本] [分区2-主副本]
   ```

2. **故障恢复**：

   - 主副本（Leader）负责读写
   - 从副本（Follower）同步主副本数据
   - 主副本故障时，从副本自动提升为主副本

3. **数据一致性**：
   - 生产者等待所有副本确认后才认为写入成功
   - 保证数据不丢失

#### 副本数的配置

```yaml
# docker-compose.yml 中的单机配置
kafka:
  environment:
    KAFKA_DEFAULT_REPLICATION_FACTOR: 1 # 单机只能设为1
    KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
```

**选择原则**：

- **单机开发**：副本数 = 1
- **生产环境**：副本数 = 3（推荐）
- **高可用场景**：副本数 = 5
- **限制**：副本数不能超过 Broker 数量

### 在本项目中的应用

#### 针对 IM 场景的优化建议

1. **消息投递主题** (`im.message.deliver`)：

   ```bash
   # 创建主题时指定分区数和副本数
   kafka-topics --create \
     --topic im.message.deliver \
     --partitions 6 \
     --replication-factor 3 \
     --bootstrap-server localhost:9092
   ```

2. **分区策略**：
   ```go
   // 使用 conversation_id 作为分区键
   // 保证同一会话的消息有序
   func (s *MessageExtService) publishToKafka(convID, payload string) {
       key := []byte(convID) // conversation_id 作为 key
       s.kafka.Publish(ctx, topic, key, []byte(payload))
   }
   ```

### 性能影响

| 配置项         | 对性能的影响               |
| -------------- | -------------------------- |
| **分区数过少** | 限制并发度，成为性能瓶颈   |
| **分区数过多** | 增加内存开销，影响选举性能 |
| **副本数过少** | 数据安全风险，但性能较好   |
| **副本数过多** | 写入延迟增加，网络开销大   |

对于本项目的 IM 场景：

- **开发环境**: 分区数=3, 副本数=1
- **生产环境**: 分区数=6-12, 副本数=3
- **关键**: 使用 `conversation_id` 作为分区键，保证会话内消息有序

## WriteMessages 阻塞机制详解

### "所有消息写入完成或达到最大重试次数"的含义

当调用 `WriteMessages()` 时，该方法会一直阻塞（停在那里等待），直到满足以下**任意一个**条件：

#### 条件 1：所有消息写入完成 ✅

```go
// 成功场景
messages := []kafka.Message{
    {Topic: "im.message.deliver", Value: []byte("msg1")},
    {Topic: "im.message.deliver", Value: []byte("msg2")},
    {Topic: "im.message.deliver", Value: []byte("msg3")},
}

// 这个调用会阻塞，直到所有3条消息都成功写入Kafka
err := writer.WriteMessages(ctx, messages...)
if err == nil {
    // 说明所有消息都成功写入
    fmt.Println("所有消息写入成功")
}
```

#### 条件 2：达到最大重试次数 ❌

```go
// 失败场景
err := writer.WriteMessages(ctx, messages...)
if err != nil {
    // 可能的错误原因：
    // - Kafka broker 不可达
    // - 网络超时
    // - 分区不存在
    // - 认证失败等

    fmt.Printf("写入失败: %v\n", err)
}
```

### 重试机制的工作原理

#### 重试过程示例

```
第1次尝试: 发送消息 -> 网络超时 -> 失败
第2次尝试: 发送消息 -> 连接拒绝 -> 失败
第3次尝试: 发送消息 -> 分区leader选举中 -> 失败
达到最大重试次数(3次) -> 返回错误，停止阻塞
```

### 在本项目中的实际表现

#### 当前 Outbox 分发器的使用

```go
// cmd/outbox/main.go 当前实现
for _, r := range rows {
    var key json.RawMessage

    // 这里会阻塞直到成功或达到最大重试次数
    _ = producer.Publish(ctx, producer.Topic(r.Topic), key, r.Payload)
    //  ^-- 注意：当前代码忽略了错误！

    // 无论成功失败都标记为已发送（这可能有问题）
    if err := q.MarkOutboxEventSent(ctx, r.ID); err != nil {
        log.Printf("mark sent failed id=%d: %v", r.ID, err)
    }
}
```

#### 改进版本（处理重试失败）

```go
// 改进后的 Outbox 分发器
for _, r := range rows {
    var key json.RawMessage

    // 正确处理发布结果
    err := producer.Publish(ctx, producer.Topic(r.Topic), key, r.Payload)
    if err != nil {
        log.Printf("publish failed for event %d: %v", r.ID, err)

        // 标记为失败，下次继续重试
        if markErr := q.MarkOutboxEventFailed(ctx, r.ID); markErr != nil {
            log.Printf("mark failed error: %v", markErr)
        }
        continue
    }

    // 只有成功时才标记为已发送
    if err := q.MarkOutboxEventSent(ctx, r.ID); err != nil {
        log.Printf("mark sent failed id=%d: %v", r.ID, err)
    }
}
```

### 常见的重试失败场景

#### 网络问题

```
时间线：
10:00:00 - 第1次尝试发送消息
10:00:02 - 网络超时，准备重试
10:00:05 - 第2次尝试发送消息
10:00:07 - 网络超时，准备重试
10:00:10 - 第3次尝试发送消息
10:00:12 - 网络超时，达到最大重试次数
10:00:12 - WriteMessages 返回错误，结束阻塞
```

### 总结

- **阻塞等待**：`WriteMessages` 会一直等待，不会立即返回
- **成功条件**：所有消息都成功写入 Kafka 并得到确认
- **失败条件**：经过多次重试后仍然失败，达到配置的最大重试次数
- **本项目影响**：当前 Outbox 分发器没有正确处理重试失败的情况，建议改进错误处理逻辑

这种设计保证了消息的可靠投递，同时通过重试机制处理临时性故障，是 Kafka 客户端的标准行为。
