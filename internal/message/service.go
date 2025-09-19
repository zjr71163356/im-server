package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"im-server/pkg/broker"
	"im-server/pkg/config"
	"im-server/pkg/dao"
	"im-server/pkg/protocol/pb/messagepb"
	mongostore "im-server/pkg/storage/mongo"

	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MessageExtService 消息服务
type MessageExtService struct {
	messagepb.UnimplementedMessageExtServiceServer
	queries dao.Querier
	rdb     redis.Cmdable
	mongo   *mongostore.Client
	kafka   *broker.KafkaProducer
}

// NewMessageExtService 创建一个新的 MessageExtService 实例
func NewMessageExtService(queries dao.Querier, rdb redis.Cmdable, mongo *mongostore.Client) *MessageExtService {
	return &MessageExtService{
		queries: queries,
		rdb:     rdb,
		mongo:   mongo,
		kafka:   broker.NewKafkaProducer(config.Config.Broker),
	}
}

// SendMessage 发送单聊消息
func (s *MessageExtService) SendMessage(ctx context.Context, req *messagepb.SendMessageRequest) (*messagepb.SendMessageReply, error) {
	// 1. 获取用户身份
	uid, ok1 := ctx.Value("user_id").(uint64)
	did, ok2 := ctx.Value("device_id").(uint64)
	if !ok1 || !ok2 {
		return nil, status.Error(codes.Unauthenticated, "missing identity")
	}

	// 2. 参数校验
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// 3. 幂等
	idKey := fmt.Sprintf("msg:%d:%d:%s", uid, did, req.ClientMsgId)
	if cached := s.rdb.Get(ctx, idKey).Val(); cached != "" {
		var resp messagepb.SendMessageReply
		_ = json.Unmarshal([]byte(cached), &resp)
		return &resp, nil
	}

	// 4. 校验好友关系
	cnt, err := s.queries.CheckFriendship(ctx, dao.CheckFriendshipParams{UserID: uid, FriendID: req.RecipientId})
	if err != nil || cnt == 0 {
		return nil, status.Error(codes.PermissionDenied, "not friends")
	}

	// 5. 会话ID（单聊：min_uid_max_uid）
	convID := buildP2PConvID(uid, req.RecipientId)

	// 6. 序列号（Redis 递增）
	seq := s.rdb.Incr(ctx, fmt.Sprintf("conv_seq:%s", convID)).Val()

	// 7. 组装并持久化（Mongo 消息体 + MySQL 索引 + Outbox）
	resp, err := s.storeMessageWithOutbox(ctx, uid, req, convID, seq)
	if err != nil {
		return nil, err
	}

	// 8. 写入幂等缓存
	b, _ := json.Marshal(resp)
	_ = s.rdb.Set(ctx, idKey, string(b), 24*time.Hour).Err()
	return resp, nil
}

func buildP2PConvID(a, b uint64) string {
	if a < b {
		return fmt.Sprintf("p_%d_%d", a, b)
	}
	return fmt.Sprintf("p_%d_%d", b, a)
}

// storeMessageWithOutbox 将消息体写入 Mongo，并在一个 DB 事务内写入索引/会话/未读；Outbox 在 Mongo 成功后立即写入
func (s *MessageExtService) storeMessageWithOutbox(ctx context.Context, senderID uint64, req *messagepb.SendMessageRequest, convID string, seq int64) (*messagepb.SendMessageReply, error) {
	// 生成 message_id（简单用时间+seq，可换为雪花）
	msgID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), seq)

	// 1) 保存 Mongo 消息体
	contentType := inferContentType(req.GetContent())
	bodyRaw, _ := json.Marshal(req.GetContent())
	if err := s.mongo.SaveMessageBody(ctx, &mongostore.MessageBody{
		MessageID:      msgID,
		ConversationID: convID,
		SenderID:       senderID,
		RecipientID:    req.RecipientId,
		Type:           contentType,
		Body:           bodyRaw,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "save body: %v", err)
	}

	// 1.5) 立即写入 Outbox（与后续 MySQL 事务解耦，用于失败补偿与异步投递）
	payload, _ := json.Marshal(map[string]any{
		"message_id":      msgID,
		"conversation_id": convID,
		"seq":             seq,
		"sender_id":       senderID,
		"recipient_id":    req.RecipientId,
		"type":            contentType,
	})
	if err := s.queries.InsertOutboxEvent(ctx, dao.InsertOutboxEventParams{Topic: "message.deliver", Payload: payload}); err != nil {
		// 不阻断主流程：记录日志，后续仍尝试 MySQL 事务与即时发布
		log.Printf("warn: insert outbox failed (will continue): %v", err)
	}

	// 2) MySQL 事务：索引/会话/未读
	tx, ok := s.queries.(*dao.Queries) // 需要底层 *Queries 才能开启事务
	if !ok {
		return nil, status.Error(codes.Internal, "queries type not *dao.Queries")
	}
	// 从 *Queries 提取底层 DBTX 为 *sql.DB
	dbtx, ok := any(tx).(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return nil, status.Error(codes.Internal, "db does not support transactions")
	}
	sqlTx, err := dbtx.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	q := tx.WithTx(sqlTx)
	defer func() {
		if err != nil {
			_ = sqlTx.Rollback()
		}
	}()

	// Upsert 会话
	if err = q.UpsertConversationOnSend(ctx, dao.UpsertConversationOnSendParams{
		ConversationID: convID,
		JSONARRAY:      senderID,
		JSONARRAY_2:    req.RecipientId,
		LastMessageID:  sql.NullString{String: msgID, Valid: true},
		LastSeq:        sql.NullInt64{Int64: seq, Valid: true},
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "upsert conversation: %v", err)
	}
	// Upsert 双方 user_conversation
	if err = q.UpsertUserConversationOnSend(ctx, dao.UpsertUserConversationOnSendParams{UserID: senderID, ConversationID: convID}); err != nil {
		return nil, status.Errorf(codes.Internal, "upsert sender conv: %v", err)
	}
	if err = q.UpsertUserConversationOnSend(ctx, dao.UpsertUserConversationOnSendParams{UserID: req.RecipientId, ConversationID: convID}); err != nil {
		return nil, status.Errorf(codes.Internal, "upsert recipient conv: %v", err)
	}
	// 插入消息索引
	if err = q.InsertMessageIndex(ctx, dao.InsertMessageIndexParams{
		MessageID:      msgID,
		ConversationID: convID,
		SenderID:       senderID,
		RecipientID:    req.RecipientId,
		MessageType:    int8(contentType),
		Seq:            seq,
		Status:         sql.NullInt16{Int16: 2, Valid: true},
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "insert index: %v", err)
	}
	// 未读 +1（接收方）
	if err = q.IncrUnreadOnRecipient(ctx, dao.IncrUnreadOnRecipientParams{UserID: req.RecipientId, ConversationID: convID}); err != nil {
		return nil, status.Errorf(codes.Internal, "incr unread: %v", err)
	}

	if err = sqlTx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	// 3) 尝试立即异步投递到 Kafka（即使失败也有 Outbox 兜底/补偿）
	if err := s.kafka.Publish(ctx, s.kafka.Topic("message.deliver"), []byte(convID), payload); err != nil {
		log.Printf("kafka publish failed: %v, will rely on outbox", err)
	}

	return &messagepb.SendMessageReply{
		MessageId:      msgID,
		ConversationId: convID,
		Seq:            seq,
		ServerTime:     time.Now().UnixMilli(),
		ClientMsgId:    req.ClientMsgId,
	}, nil
}

func inferContentType(c *messagepb.MessageContent) int32 {
	switch c.GetContent().(type) {
	case *messagepb.MessageContent_Text:
		return 1
	case *messagepb.MessageContent_Image:
		return 2
	case *messagepb.MessageContent_Audio:
		return 3
	case *messagepb.MessageContent_File:
		return 5
	default:
		return 0
	}
}
