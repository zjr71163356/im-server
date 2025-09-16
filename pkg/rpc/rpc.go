package rpc

import (
	"im-server/pkg/config"
	"im-server/pkg/protocol/pb/devicepb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	deviceIntClient devicepb.DeviceIntServiceClient
)

func SetDeviceIntServiceClient(client devicepb.DeviceIntServiceClient) {
	deviceIntClient = client
}

func GetDeviceIntServiceClient() devicepb.DeviceIntServiceClient {
	if deviceIntClient == nil {
		conn := newGrpcClient(config.Config.GRPCClient.DeviceTargetAddr)
		deviceIntClient = devicepb.NewDeviceIntServiceClient(conn)
	}
	return deviceIntClient
}

func newGrpcClient(address string) *grpc.ClientConn {
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil
	}
	return conn
}
