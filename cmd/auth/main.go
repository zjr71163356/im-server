package main

import (
	"database/sql"
	"log"
	"net"

	"im-server/internal/auth"
	"im-server/pkg/config"
	"im-server/pkg/dao"
	"im-server/pkg/protocol/pb/authpb"

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

	// 创建 Auth 服务实例
	authService := auth.NewAuthIntService(queries)

	// 启动 gRPC 服务器
	listener, err := net.Listen("tcp", config.Config.Services.Auth.RPCAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	authpb.RegisterAuthIntServiceServer(grpcServer, authService)

	log.Printf("Auth service is running on %s", config.Config.Services.Auth.RPCAddr)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
