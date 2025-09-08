package main

import (
	"im-server/internal/device"
	"im-server/pkg/config"
	"im-server/pkg/protocol/pb/devicepb"
	"log/slog"
	"net"

	"google.golang.org/grpc"
)

func main() {

	server := grpc.NewServer()
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
