-- name: CreateDevice :execresult
-- 创建设备
INSERT INTO `device` (
    created_at, updated_at, user_id, type, brand, model, 
    system_version, sdk_version, status, conn_addr, client_addr
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetDevice :one
-- 根据设备ID获取设备信息
SELECT * FROM `device` 
WHERE id = ? LIMIT 1;

-- name: GetUserDevices :many
-- 获取用户的所有设备
SELECT * FROM `device` 
WHERE user_id = ?
ORDER BY updated_at DESC;

-- name: GetOnlineDevices :many
-- 获取在线设备列表
SELECT * FROM `device` 
WHERE status = 1
ORDER BY updated_at DESC;

-- name: UpdateDeviceStatus :exec
-- 更新设备在线状态
UPDATE `device` 
SET updated_at = ?, status = ?, conn_addr = ?, client_addr = ?
WHERE id = ?;

-- name: UpdateDeviceOffline :exec
-- 设置设备离线
UPDATE `device` 
SET updated_at = ?, status = 0, conn_addr = '', client_addr = ''
WHERE id = ?;

-- name: DeleteDevice :exec
-- 删除设备
DELETE FROM `device` 
WHERE id = ?;

-- name: GetDeviceByUserAndType :one
-- 根据用户ID和设备类型获取设备
SELECT * FROM `device` 
WHERE user_id = ? AND type = ? 
LIMIT 1;
