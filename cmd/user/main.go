package main

import (
	"im-server/pkg/config"
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

func main() {
	server := grpc.NewServer()

	// 注册用户服务（不包括认证功能）
	// 认证功能已迁移到 auth 服务

	listener, err := net.Listen("tcp", config.Config.Services.User.RPCAddr)
	if err != nil {
		panic(err)
	}

	slog.Info("User service starting", "addr", config.Config.Services.User.RPCAddr)
	err = server.Serve(listener)
	if err != nil {
		slog.Error("serve error", "error", err)
	}
}
