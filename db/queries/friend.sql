-- Friend Request Queries

-- name: CreateFriendRequest :exec
-- 创建好友申请
INSERT INTO `friend_request` (
    requester_id, recipient_id, message, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?
);

-- name: GetFriendRequest :one
-- 获取好友申请
SELECT * FROM `friend_request` 
WHERE id = ?
LIMIT 1;

-- name: GetFriendRequestByUsers :one
-- 根据申请人和接收人获取好友申请
SELECT * FROM `friend_request` 
WHERE requester_id = ? AND recipient_id = ?
ORDER BY created_at DESC
LIMIT 1;

-- name: GetPendingFriendRequests :many
-- 获取待处理的好友申请列表
SELECT * FROM `friend_request` 
WHERE recipient_id = ? AND status = 0
ORDER BY created_at DESC;

-- name: GetSentFriendRequests :many
-- 获取发送的好友申请列表
SELECT * FROM `friend_request` 
WHERE requester_id = ?
ORDER BY created_at DESC;

-- name: UpdateFriendRequestStatus :exec
-- 更新好友申请状态
UPDATE `friend_request` 
SET status = ?, updated_at = ?
WHERE id = ?;

-- Friend Relationship Queries

-- name: CreateFriend :exec
-- 创建好友关系
INSERT INTO `friend` (
    user_id, friend_id, remark, category_id, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?
);

-- name: GetFriend :one
-- 获取好友关系
SELECT * FROM `friend` 
WHERE user_id = ? AND friend_id = ? 
LIMIT 1;

-- name: GetUserFriends :many
-- 获取用户的所有好友
SELECT * FROM `friend` 
WHERE user_id = ? AND is_blocked = 0
ORDER BY created_at DESC;

-- name: GetUserFriendsByCategory :many
-- 根据分类获取用户好友
SELECT * FROM `friend` 
WHERE user_id = ? AND category_id = ? AND is_blocked = 0
ORDER BY created_at DESC;

-- name: UpdateFriendRemark :exec
-- 更新好友备注
UPDATE `friend` 
SET remark = ?, updated_at = ?
WHERE user_id = ? AND friend_id = ?;

-- name: UpdateFriendCategory :exec
-- 更新好友分类
UPDATE `friend` 
SET category_id = ?, updated_at = ?
WHERE user_id = ? AND friend_id = ?;

-- name: BlockFriend :exec
-- 屏蔽好友
UPDATE `friend` 
SET is_blocked = 1, updated_at = ?
WHERE user_id = ? AND friend_id = ?;

-- name: UnblockFriend :exec
-- 取消屏蔽好友
UPDATE `friend` 
SET is_blocked = 0, updated_at = ?
WHERE user_id = ? AND friend_id = ?;

-- name: DeleteFriend :exec
-- 删除好友关系
DELETE FROM `friend` 
WHERE user_id = ? AND friend_id = ?;

-- name: CheckFriendship :one
-- 检查两个用户是否是好友
SELECT COUNT(*) as is_friend FROM `friend` 
WHERE user_id = ? AND friend_id = ? AND is_blocked = 0;
