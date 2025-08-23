-- 删除用户认证相关字段
ALTER TABLE `user` 
DROP COLUMN `hashed_password`,
DROP COLUMN `salt`;
