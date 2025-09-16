package main

import (
	"database/sql"
	"log"
	"net"

	"im-server/internal/friend"
	"im-server/pkg/config"
	"im-server/pkg/dao"
	"im-server/pkg/protocol/pb/friendpb"
	"im-server/pkg/rpc"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
)

func main() {
	// 初始化数据库连接
	db, err := sql.Open("mysql", config.Config.Database.MySQL.DSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	// 创建 queries 实例
	queries := dao.New(db)

	// 创建 Friend 服务实例
	friendService := friend.NewFriendExtService(queries)

	// 启动 gRPC 服务器
	listener, err := net.Listen("tcp", config.Config.Services.Friend.RPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// 使用参数校验 + JWT 认证拦截器（链式）
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			rpc.ValidationUnaryInterceptor(),
			rpc.JWTAuthUnaryInterceptor(),
		),
	)
	friendpb.RegisterFriendExtServiceServer(grpcServer, friendService)

	log.Printf("Friend service is running on %s", config.Config.Services.Friend.RPCAddr)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
