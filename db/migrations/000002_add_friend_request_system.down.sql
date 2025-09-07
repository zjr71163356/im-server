-- 删除新的表
DROP TABLE IF EXISTS `friend`;
DROP TABLE IF EXISTS `friend_request`;

-- 重新创建原来的 friend 表结构
CREATE TABLE `friend` (
  `user_id` bigint unsigned NOT NULL COMMENT '用户id',
  `friend_id` bigint unsigned NOT NULL COMMENT '好友id',
  `created_at` datetime NOT NULL COMMENT '创建时间',
  `updated_at` datetime NOT NULL COMMENT '更新时间',
  `remarks` varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '备注',
  `extra` varchar(1024) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL COMMENT '附加属性',
  `status` tinyint NOT NULL COMMENT '状态，1：申请，2：同意',
  PRIMARY KEY (`user_id`,`friend_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='好友';