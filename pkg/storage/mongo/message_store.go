package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MessageBody struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	MessageID      string             `bson:"message_id"`
	ConversationID string             `bson:"conversation_id"`
	SenderID       uint64             `bson:"sender_id"`
	RecipientID    uint64             `bson:"recipient_id"`
	Type           int32              `bson:"type"`
	Body           []byte             `bson:"body"`
	CreatedAt      time.Time          `bson:"created_at"`
}

// EnsureIndexes 创建必要索引
func (c *Client) EnsureIndexes(ctx context.Context) error {
	_, err := c.Messages.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "message_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "conversation_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
	})
	return err
}

// SaveMessageBody 存储消息体（已按服务内校验完成）
func (c *Client) SaveMessageBody(ctx context.Context, mb *MessageBody) error {
	if mb.CreatedAt.IsZero() {
		mb.CreatedAt = time.Now()
	}
	_, err := c.Messages.InsertOne(ctx, mb)
	return err
}
