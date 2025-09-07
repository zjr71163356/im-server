-- 回滚初始化迁移
-- 删除所有表，顺序需要考虑外键依赖关系

-- 首先删除依赖其他表的表
DROP TABLE IF EXISTS `user_message`;
DROP TABLE IF EXISTS `group_user`;
DROP TABLE IF EXISTS `friend`;
DROP TABLE IF EXISTS `friend_request`;

-- 然后删除核心业务表
DROP TABLE IF EXISTS `message`;
DROP TABLE IF EXISTS `seq`;
DROP TABLE IF EXISTS `device`;
DROP TABLE IF EXISTS `group`;
DROP TABLE IF EXISTS `user`;

-- 最后删除数据库（可选，通常在生产环境中不建议删除整个数据库）
-- DROP DATABASE IF EXISTS `gim`;