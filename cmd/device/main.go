package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net"

	"im-server/internal/device"
	"im-server/pkg/config"
	"im-server/pkg/dao"
	devicepb "im-server/pkg/protocol/pb/devicepb"

	_ "github.com/go-sql-driver/mysql"
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

	db, err := sql.Open("mysql", config.Config.Database.MySQL.DSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	queries := dao.New(db)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(validationUnaryInterceptor),
	)
	// pb.RegisterConnectServiceServer(server, &connect.ConnectService{})
	listener, err := net.Listen("tcp", config.Config.Services.Device.RPCAddr)
	if err != nil {
		panic(err)
	}
	devicepb.RegisterDeviceIntServiceServer(server, device.NewDeviceIntService(queries))
	err = server.Serve(listener)
	if err != nil {
		slog.Error("serve error", "error", err)
	}
}
