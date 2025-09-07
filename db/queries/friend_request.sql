-- name: CreateFriendRequest :exec
-- 创建好友申请
INSERT INTO `friend_request` (
    requester_id, recipient_id, status, message, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?
);

-- name: GetFriendRequest :one
-- 获取指定的好友申请
SELECT * FROM `friend_request` 
WHERE id = ? 
LIMIT 1;

-- name: GetFriendRequestByUsers :one
-- 根据申请人和接收人获取好友申请
SELECT * FROM `friend_request` 
WHERE requester_id = ? AND recipient_id = ?
ORDER BY created_at DESC
LIMIT 1;

-- name: GetReceivedFriendRequests :many
-- 获取收到的好友申请列表
SELECT * FROM `friend_request` 
WHERE recipient_id = ? AND status = ?
ORDER BY created_at DESC;

-- name: GetSentFriendRequests :many
-- 获取发送的好友申请列表
SELECT * FROM `friend_request` 
WHERE requester_id = ? AND status = ?
ORDER BY created_at DESC;

-- name: GetPendingFriendRequests :many
-- 获取待处理的好友申请
SELECT * FROM `friend_request` 
WHERE recipient_id = ? AND status = 0
ORDER BY created_at DESC;

-- name: UpdateFriendRequestStatus :exec
-- 更新好友申请状态
UPDATE `friend_request` 
SET status = ?, updated_at = ?
WHERE id = ?;

-- name: AcceptFriendRequest :exec
-- 同意好友申请
UPDATE `friend_request` 
SET status = 1, updated_at = ?
WHERE id = ?;

-- name: RejectFriendRequest :exec
-- 拒绝好友申请
UPDATE `friend_request` 
SET status = 2, updated_at = ?
WHERE id = ?;

-- name: IgnoreFriendRequest :exec
-- 忽略好友申请
UPDATE `friend_request` 
SET status = 3, updated_at = ?
WHERE id = ?;

-- name: DeleteFriendRequest :exec
-- 删除好友申请
DELETE FROM `friend_request` 
WHERE id = ?;

-- name: CheckExistingRequest :one
-- 检查是否已存在好友申请
SELECT COUNT(*) as request_exists FROM `friend_request` 
WHERE requester_id = ? AND recipient_id = ? AND status = 0;
