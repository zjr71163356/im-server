-- name: CreateUserByUsername :execresult
-- 创建用户
INSERT INTO `user` (
    created_at, updated_at, username, hashed_password
) VALUES (
    sqlc.arg(created_at), sqlc.arg(updated_at), sqlc.arg(username), sqlc.arg(hashed_password)
);

SELECT * FROM `user` WHERE id = LAST_INSERT_ID();



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
SELECT id, phone_number, hashed_password FROM `user` 
WHERE phone_number = ? LIMIT 1;

-- name: UpdateUserPassword :exec
-- 更新用户密码
UPDATE `user` 
SET updated_at = ?, hashed_password = ?
WHERE id = ?;



-- name: GetUserByEmail :one
SELECT * FROM `user` 
WHERE email = ? LIMIT 1;

-- name: GetUserByUsernameForAuth :one
-- 根据用户名获取用户认证信息
SELECT id, username, hashed_password FROM `user` 
WHERE username = ? LIMIT 1;



-- name: GetUserByEmailForAuth :one
-- 根据邮箱获取用户认证信息
SELECT id, email, hashed_password FROM `user` 
WHERE email = ? LIMIT 1;

-- name: UserExistsByUsername :one
-- 检查用户名是否存在
SELECT EXISTS(SELECT 1 FROM user WHERE username = ? LIMIT 1);