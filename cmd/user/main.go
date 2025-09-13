package main

import (
	"database/sql"
	"im-server/internal/user"
	"im-server/pkg/config"
	"im-server/pkg/dao"
	
	userpb "im-server/pkg/protocol/pb/userpb"
	"im-server/pkg/rpc"
	"log/slog"
	"net"

	_ "github.com/go-sql-driver/mysql"
	"google.golang.org/grpc"
)

func main() {
	// 初始化数据库连接
	db, err := sql.Open("mysql", config.Config.Database.MySQL.DSN)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	queries := dao.New(db)

	// 使用带拦截器的 gRPC 服务器
	server := grpc.NewServer(
		grpc.UnaryInterceptor(rpc.ValidationUnaryInterceptor()),
	)

	// 注册用户服务（不包括认证功能）
	userpb.RegisterUserExtServiceServer(server, user.NewUserService(queries))

	listener, err := net.Listen("tcp", config.Config.Services.User.RPCAddr)
	if err != nil {
		panic(err)
	}

	slog.Info("User service starting", "addr", config.Config.Services.User.RPCAddr)
	if err := server.Serve(listener); err != nil {
		slog.Error("serve error", "error", err)
	}
}
