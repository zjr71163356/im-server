-- name: InsertMessageIndex :exec
INSERT INTO message_index (
  message_id, conversation_id, sender_id, recipient_id, message_type, seq, reply_to_msg_id, status
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: GetConversationMessages :many
SELECT * FROM message_index
WHERE conversation_id = ?
ORDER BY seq DESC
LIMIT ? OFFSET ?;

-- name: GetUnreadByUserAndConversation :one
SELECT unread_count FROM user_conversation
WHERE user_id = ? AND conversation_id = ?;

-- name: UpsertUserConversationOnSend :exec
INSERT INTO user_conversation (user_id, conversation_id, last_read_seq, unread_count, is_muted, is_pinned)
VALUES (?, ?, 0, 0, 0, 0)
ON DUPLICATE KEY UPDATE updated_at = CURRENT_TIMESTAMP;

-- name: IncrUnreadOnRecipient :exec
UPDATE user_conversation
SET unread_count = unread_count + 1, updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND conversation_id = ?;

-- name: MarkRead :exec
UPDATE user_conversation
SET last_read_seq = GREATEST(last_read_seq, ?), unread_count = 0, updated_at = CURRENT_TIMESTAMP
WHERE user_id = ? AND conversation_id = ?;

-- name: UpsertConversationOnSend :exec
INSERT INTO conversation (conversation_id, type, participants, last_message_id, last_seq)
VALUES (?, 1, JSON_ARRAY(?, ?), ?, ?)
ON DUPLICATE KEY UPDATE last_message_id = VALUES(last_message_id), last_seq = VALUES(last_seq), updated_at = CURRENT_TIMESTAMP;
