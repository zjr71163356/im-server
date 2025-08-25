package main

import (
	"im-server/internal/user"
	"im-server/pkg/config"
	"im-server/pkg/protocol/pb/userpb"
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

func main() {

	server := grpc.NewServer()
	// pb.RegisterConnectServiceServer(server, &connect.ConnectService{})
	listener, err := net.Listen("tcp", config.Config.Services.User.RPCAddr)
	if err != nil {
		panic(err)
	}
	userpb.RegisterUserIntServiceServer(server, &user.UserIntService{})
	err = server.Serve(listener)
	if err != nil {
		slog.Error("serve error", "error", err)
	}
}
