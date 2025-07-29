package rpc

import (
	"context"
	"im-server/pkg/config"
	"im-server/pkg/protocol/pb/logicpb"
	"im-server/pkg/protocol/pb/userpb"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	deviceIntClient logicpb.DeviceIntServiceClient
	userIntClient   userpb.UserIntServiceClient
)

func NewDeviceIntServiceClient(cfg config.RPCClientConfig) (logicpb.DeviceIntServiceClient, *grpc.ClientConn, error) {
	_, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()
	// grpc.NewClient 需要 Go 1.59+，参数与 Dial 类似
	conn, err := grpc.NewClient(cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, err
	}
	client := logicpb.NewDeviceIntServiceClient(conn)
	return client, conn, nil
}

// 使用示例
func main() {
	cfg := config.RPCClientConfig{
		Address:     "localhost:50051",
		DialTimeout: 5 * time.Second,
	}
	client, conn, err := NewDeviceIntServiceClient(cfg)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	_, err = client.ConnSignIn(ctx, &logicpb.ConnSignInRequest{UserId: 1, DeviceId: 2})
	if err != nil {
		// 错误处理
	}
}
