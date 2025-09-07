-- 删除旧的 friend 表
DROP TABLE IF EXISTS `friend`;

-- 创建好友申请表
CREATE TABLE `friend_request` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `requester_id` bigint unsigned NOT NULL COMMENT '申请人用户ID',
  `recipient_id` bigint unsigned NOT NULL COMMENT '接收人用户ID',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '状态：0-待处理，1-已同意，2-已拒绝，3-已忽略',
  `message` varchar(255) NOT NULL DEFAULT '' COMMENT '验证消息',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_recipient_status` (`recipient_id`, `status`) COMMENT '查询待处理申请',
  KEY `idx_requester_recipient` (`requester_id`, `recipient_id`) COMMENT '防重复申请查询',
  KEY `idx_created_at` (`created_at`) COMMENT '按时间排序'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='好友申请表';

-- 创建好友关系表
CREATE TABLE `friend` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `friend_id` bigint unsigned NOT NULL COMMENT '好友用户ID',
  `remark` varchar(50) NOT NULL DEFAULT '' COMMENT '好友备注',
  `category_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '好友分类ID，0为默认分组',
  `is_blocked` tinyint NOT NULL DEFAULT '0' COMMENT '是否屏蔽：0-否，1-是',
  `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '添加时间',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_friend` (`user_id`, `friend_id`) COMMENT '防止重复好友关系',
  KEY `idx_user_id` (`user_id`) COMMENT '查询用户好友列表',
  KEY `idx_friend_id` (`friend_id`) COMMENT '反向查询',
  KEY `idx_category` (`category_id`) COMMENT '按分类查询好友'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='好友关系表';