-- 添加用户认证相关字段
ALTER TABLE `user` 
ADD COLUMN `hashed_password` varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '哈希后的密码',
ADD COLUMN `salt` varchar(32) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL DEFAULT '' COMMENT '密码盐值';

-- phone_number字段在000001_init.up.sql中已经是UNIQUE的，所以这里不需要再次添加
