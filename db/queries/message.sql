-- name: CreateMessage :execresult
-- 创建消息
INSERT INTO `message` (
    created_at, updated_at, request_id, code, content, status
) VALUES (
    ?, ?, ?, ?, ?, ?
);

-- name: GetMessage :one
-- 根据消息ID获取消息
SELECT * FROM `message` 
WHERE id = ? LIMIT 1;

-- name: UpdateMessageStatus :exec
-- 更新消息状态（如撤回消息）
UPDATE `message` 
SET updated_at = ?, status = ?
WHERE id = ?;

-- name: DeleteMessage :exec
-- 删除消息
DELETE FROM `message` 
WHERE id = ?;

-- name: CreateUserMessage :exec
-- 创建用户消息关联
INSERT INTO `user_message` (
    user_id, seq, created_at, updated_at, message_id
) VALUES (
    ?, ?, ?, ?, ?
);

-- name: GetUserMessage :one
-- 获取用户消息
SELECT * FROM `user_message` 
WHERE user_id = ? AND seq = ? 
LIMIT 1;

-- name: GetUserMessages :many
-- 获取用户消息列表
SELECT um.*, m.request_id, m.code, m.content, m.status, m.created_at as message_created_at
FROM `user_message` um
JOIN `message` m ON um.message_id = m.id
WHERE um.user_id = ? AND um.seq > ?
ORDER BY um.seq ASC
LIMIT ?;

-- name: GetUserLatestSeq :one
-- 获取用户最新的消息序列号
SELECT COALESCE(MAX(seq), 0) as latest_seq FROM `user_message` 
WHERE user_id = ?;

-- name: DeleteUserMessage :exec
-- 删除用户消息关联
DELETE FROM `user_message` 
WHERE user_id = ? AND seq = ?;
