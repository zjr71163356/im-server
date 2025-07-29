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
	Config          config.RPCClientConfig
)

func init() {
	Config = NewConfig()
}

func NewConfig() config.RPCClientConfig {
	return config.RPCClientConfig{
		Address: "127.0.0.1:8000",
	}
}

func GetDeviceIntServiceClient() logicpb.DeviceIntServiceClient {

	if deviceIntClient == nil {
		// grpc.NewClient 需要 Go 1.59+，参数与 Dial 类似
		conn := newGrpcClient(Config)
		deviceIntClient = logicpb.NewDeviceIntServiceClient(conn)

	}

	return deviceIntClient
}

func newGrpcClient(cfg config.RPCClientConfig) *grpc.ClientConn {
	// grpc.NewClient 需要 Go 1.59+，参数与 Dial 类似
	conn, err := grpc.NewClient(cfg.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		return nil
	}
	return conn
}
