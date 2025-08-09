package device

import (
	"context"
	"fmt"
	"im-server/internal/repo"
	"im-server/pkg/protocol/pb/logicpb"
	"im-server/pkg/protocol/pb/userpb"
	"im-server/pkg/rpc"

	"google.golang.org/protobuf/types/known/emptypb"
)

// DeviceIntService 模拟设备服务
// 继承 gRPC 的 DeviceIntServiceServer 接口
// 以便实现设备相关的业务逻辑。
// 这里的逻辑可以包括设备登录、状态更新等功能。

type DeviceIntService struct {
	logicpb.UnsafeDeviceIntServiceServer
	queries *repo.Queries
}

func NewDeviceIntService(queries *repo.Queries) *DeviceIntService {
	return &DeviceIntService{queries: queries}
}

//为了在ConnSignIn中能够使用repo包中访问数据库的方法
//在DeviceIntService结构体中添加queries字段
//通过函数的receiver访问queries再访问数据库函数

func (s *DeviceIntService) ConnSignIn(ctx context.Context, req *logicpb.ConnSignInRequest) (*emptypb.Empty, error) {
	_, err := rpc.GetUserIntServiceClient().Auth(ctx, &userpb.AuthRequest{
		UserId:   req.UserId,
		DeviceId: req.DeviceId,
		Token:    req.Token,
	})
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %v", err)
	}
	device, err := s.queries.GetDevice(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %v", err)
	}
	err = SetDeviceOnline(ctx, device)
	if err != nil {
		return nil, fmt.Errorf("failed to set device online: %v", err)
	}

	// TODO: 实现登录逻辑
	return new(emptypb.Empty), nil
}
