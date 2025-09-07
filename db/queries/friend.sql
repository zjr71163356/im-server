-- name: CreateFriend :exec
-- 创建好友关系
INSERT INTO `friend` (
    user_id, friend_id, remark, category_id, is_blocked, created_at, updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
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
-- 按分类获取用户的好友
SELECT * FROM `friend` 
WHERE user_id = ? AND category_id = ? AND is_blocked = 0
ORDER BY created_at DESC;

-- name: GetBlockedFriends :many
-- 获取被屏蔽的好友
SELECT * FROM `friend` 
WHERE user_id = ? AND is_blocked = 1
ORDER BY created_at DESC;

-- name: UpdateFriendRemark :exec
-- 更新好友备注
UPDATE `friend` 
SET updated_at = ?, remark = ?
WHERE user_id = ? AND friend_id = ?;

-- name: UpdateFriendCategory :exec
-- 更新好友分类
UPDATE `friend` 
SET updated_at = ?, category_id = ?
WHERE user_id = ? AND friend_id = ?;

-- name: BlockFriend :exec
-- 屏蔽好友
UPDATE `friend` 
SET updated_at = ?, is_blocked = 1
WHERE user_id = ? AND friend_id = ?;

-- name: UnblockFriend :exec
-- 取消屏蔽好友
UPDATE `friend` 
SET updated_at = ?, is_blocked = 0
WHERE user_id = ? AND friend_id = ?;

-- name: DeleteFriend :exec
-- 删除好友关系
DELETE FROM `friend` 
WHERE user_id = ? AND friend_id = ?;

-- name: CheckFriendship :one
-- 检查两个用户是否是好友
SELECT COUNT(*) as is_friend FROM `friend` 
WHERE user_id = ? AND friend_id = ?;
