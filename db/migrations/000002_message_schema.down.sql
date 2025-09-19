-- Revert message indexing, conversations, and outbox schema

DROP TABLE IF EXISTS `outbox_events`;
DROP TABLE IF EXISTS `user_conversation`;
DROP TABLE IF EXISTS `conversation`;
DROP TABLE IF EXISTS `message_index`;
