package main

import (
	"context"
	"im-server/internal/device"
	"im-server/pkg/config"
	devicepb "im-server/pkg/protocol/pb/devicepb"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// validationUnaryInterceptor 在所有 unary RPC 上统一调用生成的 Validate() 方法（如果存在）
func validationUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if v, ok := req.(interface{ Validate() error }); ok {
		if err := v.Validate(); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
		}
	}
	return handler(ctx, req)
}

func main() {

	server := grpc.NewServer(
		grpc.UnaryInterceptor(validationUnaryInterceptor),
	)
	// pb.RegisterConnectServiceServer(server, &connect.ConnectService{})
	listener, err := net.Listen("tcp", config.Config.Services.Device.RPCAddr)
	if err != nil {
		panic(err)
	}
	devicepb.RegisterDeviceIntServiceServer(server, &device.DeviceIntService{})
	err = server.Serve(listener)
	if err != nil {
		slog.Error("serve error", "error", err)
	}
}
