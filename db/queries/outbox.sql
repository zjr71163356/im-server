-- name: InsertOutboxEvent :exec
INSERT INTO outbox_events (topic, payload, status, retry_count, next_delivery_at)
VALUES (?, ?, 'pending', 0, NULL);

-- name: GetPendingOutboxEvents :many
SELECT id, topic, payload FROM outbox_events
WHERE status = 'pending' AND (next_delivery_at IS NULL OR next_delivery_at <= CURRENT_TIMESTAMP)
ORDER BY id ASC
LIMIT ?;

-- name: MarkOutboxEventSent :exec
UPDATE outbox_events SET status = 'sent', updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: MarkOutboxEventFailed :exec
UPDATE outbox_events SET status = 'failed', retry_count = retry_count + 1, next_delivery_at = DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 5 SECOND)
WHERE id = ?;
