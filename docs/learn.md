## 数据库设计：friend_request、friend 表结构设计与迁移应该怎么做?friend_request要存储到数据库吗？存储到redis会不会更好？
### `friend_request` 应该存储在哪里？数据库 vs Redis

**结论先行：强烈建议将 `friend_request` 存储在数据库（MySQL）中，而不是 Redis。**

#### 为什么存储在数据库更好？

1.  **数据持久性 (Durability)**：好友请求是重要的用户交互数据。如果 Redis 服务器因为故障、重启而没有配置好持久化（或持久化失败），所有未处理的好友请求都会丢失。这对用户体验是毁灭性的。MySQL 作为关系型数据库，其首要设计目标就是保证数据的持久性和可靠性。

2.  **事务一致性 (Transactional Consistency)**：当用户“同意”一个好友请求时，通常需要执行两个操作：
    *   更新 `friend_request` 表的状态为 `accepted`。
    *   在 `friend` 表中**双向插入**两条好友关系记录。
    将这些操作放在一个数据库事务中，可以保证它们要么全部成功，要么全部失败。如果使用 Redis + MySQL，你需要处理分布式事务的复杂性，很容易出现数据不一致（例如，请求在 Redis 中删了，但好友关系在 MySQL 中没建成功）。

3.  **数据完整性 (Data Integrity)**：通过外键约束，你可以确保 `friend_request` 中的 `requester_id` 和 `recipient_id` 必须是 `user` 表中存在的用户，防止产生无效的“幽灵”请求。Redis 无法提供这种关系约束。

4.  **查询与分析能力 (Querying & Analytics)**：未来你可能需要分析好友请求数据（例如，用户的社交活跃度、请求拒绝率等）。SQL 提供了强大的查询能力，而 Redis 在这方面非常受限。

#### Redis 的适用场景

Redis 在这个场景中可以作为**辅助缓存**，而不是主存储：
*   **缓存未读请求数**：当用户登录时，可以将其未读好友请求的数量缓存到 Redis。这样，每次前端需要显示“小红点”时，直接从 Redis 读取，而不用频繁查询数据库。当请求被处理后，再更新缓存。
*   **实现“24小时内只能申请一次”**：这个功能用 Redis 的 `SET` 命令配合 `EX` (过期时间) 非常完美。当用户 A 向 B 发送请求时，你可以在 Redis 中设置一个键，如 `friend_req_lock:A_to_B`，并设置 24 小时过期。下次再申请时，先检查这个键是否存在即可。

---

### 数据库设计与迁移步骤

下面是具体的设计和操作流程，完全符合你项目的 `golang-migrate` 和 `sqlc` 工作流。

#### 第一步：设计表结构 (`friend_request` 和 `friend`)

1.  **`friend_request` 表**：用于存储好友申请的记录。

    | 列名 | 类型 | 约束/注释 |
    | :--- | :--- | :--- |
    | `id` | `BIGINT` | `PRIMARY KEY`, `AUTO_INCREMENT` |
    | `requester_id` | `BIGINT` | `NOT NULL`, `FOREIGN KEY (user.id)` - 申请人ID |
    | `recipient_id` | `BIGINT` | `NOT NULL`, `FOREIGN KEY (user.id)` - 接收人ID |
    | `status` | `TINYINT` | `NOT NULL`, `DEFAULT 0` - 状态 (0: pending, 1: accepted, 2: rejected) |
    | `message` | `VARCHAR(255)` | `DEFAULT ''` - 验证消息 |
    | `created_at` | `TIMESTAMP` | `NOT NULL`, `DEFAULT CURRENT_TIMESTAMP` |
    | `updated_at` | `TIMESTAMP` | `NOT NULL`, `DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP` |

    **索引建议**：
    *   在 `recipient_id` 和 `status` 上创建复合索引 `(recipient_id, status)`，用于快速查询某人的待处理请求列表。
    *   在 `requester_id` 和 `recipient_id` 上创建唯一索引 `(requester_id, recipient_id)`，可以防止在数据库层面重复发送（但业务逻辑层面用 Redis 控制频率更好）。

2.  **`friend` 表**：用于存储已建立的好友关系（双向）。

    | 列名 | 类型 | 约束/注释 |
    | :--- | :--- | :--- |
    | `id` | `BIGINT` | `PRIMARY KEY`, `AUTO_INCREMENT` |
    | `user_id` | `BIGINT` | `NOT NULL`, `FOREIGN KEY (user.id)` - 用户ID |
    | `friend_id` | `BIGINT` | `NOT NULL`, `FOREIGN KEY (user.id)` - 好友的用户ID |
    | `remark` | `VARCHAR(255)` | `DEFAULT ''` - 备注名 |
    | `created_at` | `TIMESTAMP` | `NOT NULL`, `DEFAULT CURRENT_TIMESTAMP` |

    **索引建议**：
    *   在 `user_id` 上创建索引，用于快速获取某人的好友列表。
    *   在 `(user_id, friend_id)` 上创建唯一索引，防止重复添加好友。

#### 第二步：创建迁移文件

根据你的项目结构，在 migrations 目录下创建新的迁移文件。假设下一个版本号是 `000003`。

1.  **创建 `000003_create_friend_tables.up.sql` 文件**：

    ````sql
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
    ````

2.  **创建 `000003_create_friend_tables.down.sql` 文件**：

    ````sql
    -- filepath: /home/tyrfly/im-server/db/migrations/000003_create_friend_tables.down.sql
    DROP TABLE IF EXISTS `friend_request`;
    DROP TABLE IF EXISTS `friend`;
    ````

#### 第三步：应用数据库迁移

在你的项目根目录下运行 `make` 命令：

```bash
make migrate-up
```

这将会执行 `up` 脚本，在你的 MySQL 数据库中创建这两个表。

#### 第四步：定义 SQLC 查询

在 queries 目录下创建 `friend.sql` 和 `friend_request.sql` 文件，定义数据访问操作。

1.  **创建 `db/queries/friend_request.sql`**

    ````sql
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
    ````

2.  **创建 friend.sql**

    ````sql
    -- name: AddFriend :exec
    INSERT INTO friend (user_id, friend_id)
    VALUES (?, ?);

    -- name: ListFriends :many
    SELECT * FROM friend
    WHERE user_id = ?;

    -- name: DeleteFriend :exec
    DELETE FROM friend
    WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?);
    ````

#### 第五步：生成 Go 代码

最后，运行 `sqlc` 生成 Go 代码：

```bash
make sqlc-generate
```

现在，你就可以在你的 Go 业务逻辑代码中，通过 dao 调用这些新生成的、类型安全的方法来操作好友和好友请求了。这个流程完全符合你项目的既定规范。

## `protoc-gen-validate` 的安装
好的，我们来总结一下 `protoc-gen-validate` 的整个安装、调试过程，以及最终的解决方案。

### 目标

我们的目标是在 gRPC 服务层实现自动的请求参数校验。具体来说，是利用 `protoc-gen-validate` 插件，根据 `.proto` 文件中定义的校验规则（如字段长度、范围等），自动生成 Go 代码中的 `Validate()` 方法。这样，通过 gRPC 拦截器调用此方法，就能在业务逻辑执行前拒绝无效请求，避免像“超长用户名”那样导致数据库写入失败而产生 500 错误。

---

### 安装与调试过程中的问题

我们在安装和配置过程中遇到了一系列环环相扣的问题：

1.  **问题一：`go install` 路径错误**
    *   **尝试的命令**：`go install github.com/envoyproxy/protoc-gen-validate/protoc-gen-validate@latest`
    *   **遇到的错误**：`...module ... does not contain package...`
    *   **原因**：我们使用了旧的包安装路径。`protoc-gen-validate` 项目的目录结构已经更新，其 Go 插件的 `main` 包不再位于项目根目录，而是移到了 `cmd/protoc-gen-validate-go` 子目录中。

2.  **问题二：`protoc` 插件名称不匹配**
    *   **解决问题一后**：我们使用正确的路径 `go install .../cmd/protoc-gen-validate-go@latest` 成功安装了插件，但这在 `GOPATH/bin` 中生成的是名为 `protoc-gen-validate-go` 的可执行文件。
    *   **遇到的错误**：当运行 `make proto` 时，`protoc` 报错 `protoc-gen-validate: program not found...`。
    *   **原因**：`protoc` 在处理 `--validate_out` 标志时，默认会去 `PATH` 中寻找名为 `protoc-gen-validate` 的可执行文件，而我们安装的却是 `protoc-gen-validate-go`，两者名称不匹配。

3.  **问题三：创建符号链接时文件已存在**
    *   **尝试的命令**：为了解决名称不匹配问题，我们尝试创建一个符号链接 `ln -s .../protoc-gen-validate-go .../protoc-gen-validate`。
    *   **遇到的错误**：`ln: ... File exists`。
    *   **原因**：之前可能存在一个不正确或不完整的安装残留，导致 `protoc-gen-validate` 这个文件或链接已经存在。

4.  **问题四：`protoc` 插件参数格式错误**
    *   **解决问题三后**：我们使用 `ln -sf` 强制创建了符号链接，并修改 Makefile 添加了 `--validate_out=lang=go:.` 参数。
    *   **遇到的错误**：`[validator]unknown 'lang' parameter`。
    *   **原因**：这是最关键的一步。这个错误表明我们使用的插件是**新版本**的，而 Makefile 中的 `--validate_out=lang=go:.` 是**旧版本**的参数格式。新版插件不再接受 `lang=go` 这种形式的参数。

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
    *   **注意**：采用这种方式后，就不再需要创建 `protoc-gen-validate` 的符号链接了，因为 `protoc` 会根据 `--validate-go_out` 自动寻找 `protoc-gen-validate-go`。

3.  **重新生成代码**：
    执行 `make proto`。

    ```bash
    make proto
    ```

完成以上步骤后，`protoc` 成功调用了新版插件，并在 authpb 目录下生成了包含 `Validate()` 方法的 auth.int.pb.go 和 `auth.int.pb.validate.go` 文件，从而彻底解决了问题。现在，gRPC 拦截器可以正常工作，对超长用户名等非法请求返回 `400 Bad Request`。