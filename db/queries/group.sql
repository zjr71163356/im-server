-- name: CreateGroup :execresult
-- 创建群组
INSERT INTO `group` (
    created_at, updated_at, name, avatar_url, introduction, user_num, extra
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);

-- name: GetGroup :one
-- 根据群组ID获取群组信息
SELECT * FROM `group` 
WHERE id = ? LIMIT 1;

-- name: UpdateGroup :exec
-- 更新群组信息
UPDATE `group` 
SET updated_at = ?, name = ?, avatar_url = ?, introduction = ?, extra = ?
WHERE id = ?;

-- name: UpdateGroupUserNum :exec
-- 更新群组人数
UPDATE `group` 
SET updated_at = ?, user_num = ?
WHERE id = ?;

-- name: DeleteGroup :exec
-- 删除群组
DELETE FROM `group` 
WHERE id = ?;

-- name: ListGroups :many
-- 获取群组列表
SELECT * FROM `group` 
ORDER BY created_at DESC 
LIMIT ? OFFSET ?;

-- name: CreateGroupUser :exec
-- 添加群组成员
INSERT INTO `group_user` (
    group_id, user_id, created_at, updated_at, member_type, remarks, extra, status
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetGroupUser :one
-- 获取群组成员信息
SELECT * FROM `group_user` 
WHERE group_id = ? AND user_id = ? 
LIMIT 1;

-- name: GetGroupUsers :many
-- 获取群组所有成员
SELECT * FROM `group_user` 
WHERE group_id = ?
ORDER BY member_type ASC, created_at ASC;

-- name: GetUserGroups :many
-- 获取用户参与的所有群组
SELECT * FROM `group_user` 
WHERE user_id = ?
ORDER BY created_at DESC;

-- name: UpdateGroupUserType :exec
-- 更新群组成员类型
UPDATE `group_user` 
SET updated_at = ?, member_type = ?
WHERE group_id = ? AND user_id = ?;

-- name: DeleteGroupUser :exec
-- 移除群组成员
DELETE FROM `group_user` 
WHERE group_id = ? AND user_id = ?;
