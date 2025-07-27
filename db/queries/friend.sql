-- name: CreateFriend :exec
-- 创建好友关系
INSERT INTO `friend` (
    user_id, friend_id, created_at, updated_at, remarks, extra, status
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
WHERE user_id = ? AND status = 2
ORDER BY created_at DESC;

-- name: GetFriendRequests :many
-- 获取好友申请列表
SELECT * FROM `friend` 
WHERE friend_id = ? AND status = 1
ORDER BY created_at DESC;

-- name: UpdateFriendStatus :exec
-- 更新好友状态（同意/拒绝好友申请）
UPDATE `friend` 
SET updated_at = ?, status = ?
WHERE user_id = ? AND friend_id = ?;

-- name: UpdateFriendRemarks :exec
-- 更新好友备注
UPDATE `friend` 
SET updated_at = ?, remarks = ?
WHERE user_id = ? AND friend_id = ?;

-- name: DeleteFriend :exec
-- 删除好友关系
DELETE FROM `friend` 
WHERE user_id = ? AND friend_id = ?;

-- name: CheckFriendship :one
-- 检查两个用户是否是好友
SELECT COUNT(*) as is_friend FROM `friend` 
WHERE user_id = ? AND friend_id = ? AND status = 2;
