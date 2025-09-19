package main

import (
	"context"
	"encoding/json"
	"im-server/internal/connect"
	"im-server/pkg/config"
	"im-server/pkg/protocol/pb/connectpb"
	"im-server/pkg/rpc"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"im-server/pkg/broker"

	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
)

func main() {
	// 启动 WS 服务
	go func() {
		connect.StartWSServer(config.Config.Services.Connect.WSAddr)
	}()

	// 启动 Kafka 消费者：消费 `${prefix}.message.deliver`
	go startKafkaConsumer()

	// gRPC 服务
	server := grpc.NewServer(
		grpc.UnaryInterceptor(rpc.ValidationUnaryInterceptor()),
	)
	// pb.RegisterConnectServiceServer(server, &connect.ConnectService{})
	listener, err := net.Listen("tcp", config.Config.Services.Connect.RPCAddr)
	if err != nil {
		panic(err)
	}

	// 优雅停机
	go func() {
		if err := server.Serve(listener); err != nil {
			slog.Error("serve error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("connect shutting down...")
	server.GracefulStop()
}

func startKafkaConsumer() {
	prefix := config.Config.Broker.TopicPrefix
	topic := "message.deliver"
	if prefix != "" {
		topic = prefix + "." + topic
	}
	consumer := broker.NewKafkaConsumer(config.Config.Broker, "connect-deliver", topic)
	defer consumer.Close()

	slog.Info("connect kafka consumer starting", "topic", topic)
	ctx := context.Background()
	type deliverPayload struct {
		MessageID      string `json:"message_id"`
		ConversationID string `json:"conversation_id"`
		Seq            int64  `json:"seq"`
		SenderID       uint64 `json:"sender_id"`
		RecipientID    uint64 `json:"recipient_id"`
		Type           int32  `json:"type"`
	}

	if err := consumer.Start(ctx, func(ctx context.Context, m kafka.Message) error {
		var p deliverPayload
		if err := json.Unmarshal(m.Value, &p); err != nil {
			slog.Error("invalid payload", "err", err)
			return nil
		}
		pkt := &connectpb.Packet{
			Command:   connectpb.Command_MESSAGE,
			RequestId: 0,
			Code:      0,
			Message:   "",
			Data:      m.Value, // 直接转发 JSON，客户端按约定解析
		}
		n := connect.DeliverToUser(p.RecipientID, pkt)
		slog.Info("delivered message", "recipient", p.RecipientID, "devices", n, "seq", p.Seq)
		return nil
	}); err != nil {
		slog.Error("kafka consumer stopped", "err", err)
	}
}
