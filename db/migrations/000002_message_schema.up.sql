-- Schema upgrade: message indexing, conversations, and outbox (per docs/数据库设计说明.md)

-- 消息索引表（热数据，MySQL）
CREATE TABLE IF NOT EXISTS `message_index` (
  `message_id` VARCHAR(32) NOT NULL COMMENT '消息ID',
  `conversation_id` VARCHAR(32) NOT NULL COMMENT '会话ID',
  `sender_id` BIGINT UNSIGNED NOT NULL COMMENT '发送者ID',
  `recipient_id` BIGINT UNSIGNED NOT NULL COMMENT '接收者ID',
  `message_type` TINYINT NOT NULL COMMENT '1:文本 2:图片 3:音频 4:视频 5:文件 6:位置',
  `seq` BIGINT NOT NULL COMMENT '会话内序列号',
  `reply_to_msg_id` VARCHAR(32) DEFAULT NULL COMMENT '回复的消息ID',
  `status` TINYINT DEFAULT 2 COMMENT '消息状态',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`message_id`),
  KEY `idx_conversation_seq` (`conversation_id`, `seq` DESC),
  KEY `idx_recipient_time` (`recipient_id`, `created_at` DESC),
  KEY `idx_sender_time` (`sender_id`, `created_at` DESC),
  KEY `idx_type_time` (`message_type`, `created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='消息索引表（热数据）';


-- 会话表（热数据，MySQL）
-- 注意：MySQL 8 不支持直接为 JSON 列建索引，如需参与者索引应使用生成列。
CREATE TABLE IF NOT EXISTS `conversation` (
  `conversation_id` VARCHAR(32) NOT NULL COMMENT '会话ID',
  `type` TINYINT NOT NULL COMMENT '1:单聊 2:群聊',
  `participants` JSON NOT NULL COMMENT '参与者ID数组',
  `last_message_id` VARCHAR(32) DEFAULT NULL COMMENT '最后一条消息ID',
  `last_seq` BIGINT DEFAULT 0 COMMENT '最新序列号',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`conversation_id`),
  KEY `idx_updated_at` (`updated_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='会话表（热数据）';


-- 用户会话状态（热数据，MySQL）
CREATE TABLE IF NOT EXISTS `user_conversation` (
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `conversation_id` VARCHAR(32) NOT NULL COMMENT '会话ID',
  `last_read_seq` BIGINT DEFAULT 0 COMMENT '最后已读序列号',
  `unread_count` INT DEFAULT 0 COMMENT '未读数',
  `is_muted` TINYINT(1) DEFAULT 0 COMMENT '是否免打扰',
  `is_pinned` TINYINT(1) DEFAULT 0 COMMENT '是否置顶',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`user_id`, `conversation_id`),
  KEY `idx_user_updated` (`user_id`, `updated_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='用户会话状态';


-- Outbox事件表（热数据，MySQL）
CREATE TABLE IF NOT EXISTS `outbox_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `topic` VARCHAR(100) NOT NULL COMMENT '事件主题',
  `payload` JSON NOT NULL COMMENT '事件负载',
  `status` ENUM('pending','sent','failed') DEFAULT 'pending' COMMENT '状态',
  `retry_count` INT DEFAULT 0 COMMENT '重试次数',
  `next_delivery_at` TIMESTAMP NULL DEFAULT NULL COMMENT '下次投递时间',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_status_delivery` (`status`, `next_delivery_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='Outbox事件表';
