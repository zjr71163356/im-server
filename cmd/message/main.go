package main

import (
	"context"
	"database/sql"
	"log"
	"net"

	"im-server/internal/message"
	"im-server/pkg/config"
	"im-server/pkg/dao"
	messagepb "im-server/pkg/protocol/pb/messagepb"
	Redis "im-server/pkg/redis"
	"im-server/pkg/rpc"
	mongostore "im-server/pkg/storage/mongo"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
)

func main() {
	// MySQL
	db, err := sql.Open("mysql", config.Config.Database.MySQL.DSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	queries := dao.New(db)

	// Redis（已在包内初始化）
	rdb := Redis.RedisClient

	// Mongo
	ctx := context.Background()
	mongoCli, err := mongostore.New(ctx, config.Config.Database.Mongo)
	if err != nil {
		log.Fatalf("failed to init mongo: %v", err)
	}
	if err := mongoCli.EnsureIndexes(ctx); err != nil {
		log.Fatalf("failed to ensure mongo indexes: %v", err)
	}

	// gRPC server
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			rpc.ValidationUnaryInterceptor(),
			rpc.JWTAuthUnaryInterceptor(),
		),
	)

	msgSvc := message.NewMessageExtService(queries, rdb, mongoCli)
	messagepb.RegisterMessageExtServiceServer(server, msgSvc)

	listener, err := net.Listen("tcp", config.Config.Services.Message.RPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("Message service is running on %s", config.Config.Services.Message.RPCAddr)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
