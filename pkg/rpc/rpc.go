package rpc

import (
	"im-server/pkg/config"
	"im-server/pkg/protocol/pb/logicpb"
	"im-server/pkg/protocol/pb/userpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	deviceIntClient logicpb.DeviceIntServiceClient
	userIntClient   userpb.UserIntServiceClient
)

func GetDeviceIntServiceClient() logicpb.DeviceIntServiceClient {

	if deviceIntClient == nil {
		// grpc.NewClient 需要 Go 1.59+，参数与 Dial 类似
		conn := newGrpcClient(config.Config.GRPCClient.DeviceTargetAddr)
		deviceIntClient = logicpb.NewDeviceIntServiceClient(conn)

	}

	return deviceIntClient
}

func newGrpcClient(address string) *grpc.ClientConn {
	// grpc.NewClient 需要 Go 1.59+，参数与 Dial 类似
	// address 表示 gRPC 客户端要连接的目标服务地址
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		return nil
	}
	return conn
}
