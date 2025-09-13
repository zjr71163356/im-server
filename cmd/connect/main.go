package main

import (
	"im-server/internal/connect"
	"im-server/pkg/config"
	"im-server/pkg/rpc"
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

func main() {
	go func() {
		connect.StartWSServer(config.Config.Services.Connect.WSAddr)
	}()
	server := grpc.NewServer(
		grpc.UnaryInterceptor(rpc.ValidationUnaryInterceptor()),
	)
	// pb.RegisterConnectServiceServer(server, &connect.ConnectService{})
	listener, err := net.Listen("tcp", config.Config.Services.Connect.RPCAddr)
	if err != nil {
		panic(err)
	}

	err = server.Serve(listener)
	if err != nil {
		slog.Error("serve error", "error", err)
	}
}
