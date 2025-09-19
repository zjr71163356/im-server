package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"

	"im-server/pkg/broker"
	"im-server/pkg/config"
	"im-server/pkg/dao"

	_ "github.com/go-sql-driver/mysql"
	"github.com/segmentio/kafka-go"
)

// deliverPayload 与消息投递事件的 payload 对齐
type deliverPayload struct {
	MessageID      string `json:"message_id"`
	ConversationID string `json:"conversation_id"`
	Seq            int64  `json:"seq"`
	SenderID       uint64 `json:"sender_id"`
	RecipientID    uint64 `json:"recipient_id"`
	Type           int32  `json:"type"`
}

func main() {
	ctx := context.Background()

	// DB 连接
	db, err := sql.Open("mysql", config.Config.Database.MySQL.DSN)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()
	q := dao.New(db)

	// Kafka 消费者（订阅投递主题，做索引回填）
	prefix := config.Config.Broker.TopicPrefix
	topic := "message.deliver"
	if prefix != "" {
		topic = prefix + "." + topic
	}
	consumer := broker.NewKafkaConsumer(config.Config.Broker, "indexer-backfill", topic)
	defer consumer.Close()
	log.Printf("indexer started, consuming topic=%s", topic)

	h := func(ctx context.Context, m kafka.Message) error {
		var p deliverPayload
		if err := json.Unmarshal(m.Value, &p); err != nil {
			log.Printf("invalid payload: %v", err)
			return nil
		}

		// 开启一次性事务，确保同一条消息的回填原子性
		sqlTx, err := db.BeginTx(ctx, nil)
		if err != nil {
			log.Printf("begin tx: %v", err)
			return nil
		}
		qq := q.WithTx(sqlTx)
		rollback := func() { _ = sqlTx.Rollback() }

		// 1) 尝试插入消息索引（通过主键 message_id 幂等）
		dup := false
		if err := qq.InsertMessageIndex(ctx, dao.InsertMessageIndexParams{
			MessageID:      p.MessageID,
			ConversationID: p.ConversationID,
			SenderID:       p.SenderID,
			RecipientID:    p.RecipientID,
			MessageType:    int8(p.Type),
			Seq:            p.Seq,
			// Status 保持默认或按需设置
		}); err != nil {
			if isDuplicate(err) {
				dup = true
			} else {
				log.Printf("insert index err: %v", err)
				rollback()
				return nil
			}
		}

		// 2) Upsert 会话元信息（last_message_id/last_seq）
		if err := qq.UpsertConversationOnSend(ctx, dao.UpsertConversationOnSendParams{
			ConversationID: p.ConversationID,
			JSONARRAY:      p.SenderID,
			JSONARRAY_2:    p.RecipientID,
			LastMessageID:  sql.NullString{String: p.MessageID, Valid: true},
			LastSeq:        sql.NullInt64{Int64: p.Seq, Valid: true},
		}); err != nil {
			log.Printf("upsert conversation err: %v", err)
			rollback()
			return nil
		}

		// 3) 确保双方 user_conversation 存在
		if err := qq.UpsertUserConversationOnSend(ctx, dao.UpsertUserConversationOnSendParams{UserID: p.SenderID, ConversationID: p.ConversationID}); err != nil {
			log.Printf("upsert sender conv err: %v", err)
			rollback()
			return nil
		}
		if err := qq.UpsertUserConversationOnSend(ctx, dao.UpsertUserConversationOnSendParams{UserID: p.RecipientID, ConversationID: p.ConversationID}); err != nil {
			log.Printf("upsert recipient conv err: %v", err)
			rollback()
			return nil
		}

		// 4) 未读 +1：仅当本次确认为新插入（非重复）
		if !dup {
			if err := qq.IncrUnreadOnRecipient(ctx, dao.IncrUnreadOnRecipientParams{UserID: p.RecipientID, ConversationID: p.ConversationID}); err != nil {
				log.Printf("incr unread err: %v", err)
				rollback()
				return nil
			}
		}

		if err := sqlTx.Commit(); err != nil {
			log.Printf("commit err: %v", err)
			return nil
		}
		return nil
	}

	if err := consumer.Start(ctx, h); err != nil {
		log.Printf("consumer stopped: %v", err)
	}
}

// isDuplicate 判断是否为主键/唯一键冲突
func isDuplicate(err error) bool {
	var me mysqlError
	if errors.As(err, &me) {
		return me.Number() == 1062 // ER_DUP_ENTRY
	}
	return false
}

// 适配 go-sql-driver/mysql 错误类型（避免直接依赖具体包名）
type mysqlError interface{ Number() uint16 }
 