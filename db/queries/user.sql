-- name: CreateUser :execresult
-- 创建用户
INSERT INTO `user` (
    created_at, updated_at, phone_number, nickname, sex, avatar_url, extra, hashed_password, salt
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetUser :one
-- 根据用户ID获取用户信息
SELECT * FROM `user` 
WHERE id = ? LIMIT 1;

-- name: GetUserByPhone :one
-- 根据手机号获取用户信息
SELECT * FROM `user` 
WHERE phone_number = ? LIMIT 1;

-- name: UpdateUser :exec
-- 更新用户信息
UPDATE `user` 
SET updated_at = ?, nickname = ?, sex = ?, avatar_url = ?, extra = ?
WHERE id = ?;

-- name: UpdateUserAvatar :exec
-- 更新用户头像
UPDATE `user` 
SET updated_at = ?, avatar_url = ?
WHERE id = ?;

-- name: DeleteUser :exec
-- 删除用户
DELETE FROM `user` 
WHERE id = ?;

-- name: ListUsers :many
-- 获取用户列表
SELECT * FROM `user` 
ORDER BY created_at DESC 
LIMIT ? OFFSET ?;

-- name: GetUserByPhoneForAuth :one
-- 根据手机号获取用户认证信息
SELECT id, phone_number, hashed_password, salt FROM `user` 
WHERE phone_number = ? LIMIT 1;

-- name: UpdateUserPassword :exec
-- 更新用户密码
UPDATE `user` 
SET updated_at = ?, hashed_password = ?, salt = ?
WHERE id = ?;
