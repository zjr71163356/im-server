-- name: CreateSeq :exec
-- 创建序列号记录
INSERT INTO `seq` (
    created_at, updated_at, object_type, object_id, seq
) VALUES (
    ?, ?, ?, ?, ?
);

-- name: GetSeq :one
-- 获取序列号
SELECT * FROM `seq` 
WHERE object_type = ? AND object_id = ? 
LIMIT 1;

-- name: UpdateSeq :exec
-- 更新序列号
UPDATE `seq` 
SET updated_at = ?, seq = ?
WHERE object_type = ? AND object_id = ?;

-- name: IncrementSeq :exec
-- 递增序列号
UPDATE `seq` 
SET updated_at = ?, seq = seq + 1
WHERE object_type = ? AND object_id = ?;

-- name: GetOrCreateSeq :exec
-- 获取或创建序列号（使用 INSERT ... ON DUPLICATE KEY UPDATE）
INSERT INTO `seq` (created_at, updated_at, object_type, object_id, seq)
VALUES (?, ?, ?, ?, 1)
ON DUPLICATE KEY UPDATE 
    updated_at = VALUES(updated_at),
    seq = seq + 1;

-- name: DeleteSeq :exec
-- 删除序列号记录
DELETE FROM `seq` 
WHERE object_type = ? AND object_id = ?;
