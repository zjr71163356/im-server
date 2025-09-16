package device

import (
	"context"
	"fmt"
	"im-server/pkg/dao"
	"im-server/pkg/protocol/pb/devicepb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// DeviceIntService 模拟设备服务
// 继承 gRPC 的 DeviceIntServiceServer 接口
// 以便实现设备相关的业务逻辑。
// 这里的逻辑可以包括设备登录、状态更新等功能。

type DeviceIntService struct {
	devicepb.UnsafeDeviceIntServiceServer
	queries *dao.Queries
}

func NewDeviceIntService(queries *dao.Queries) *DeviceIntService {
	return &DeviceIntService{queries: queries}
}

//为了在ConnSignIn中能够使用dao包中访问数据库的方法
//在DeviceIntService结构体中添加queries字段
//通过函数的receiver访问queries再访问数据库函数

func (s *DeviceIntService) ConnSignIn(ctx context.Context, req *devicepb.ConnSignInRequest) (*emptypb.Empty, error) {
	// JWT 认证已在拦截器中完成，此处直接从 context 获取用户信息
	userID, ok := ctx.Value("user_id").(uint64)
	if !ok {
		return nil, fmt.Errorf("user not authenticated")
	}

	device, err := s.queries.GetDevice(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %v", err)
	}

	// 验证设备是否属于当前用户
	if device.UserID != userID {
		return nil, fmt.Errorf("device does not belong to user")
	}

	err = SetDeviceOnline(ctx, &device)
	if err != nil {
		return nil, fmt.Errorf("failed to set device online: %v", err)
	}

	// TODO: 实现登录逻辑
	return new(emptypb.Empty), nil
}

func (s *DeviceIntService) Offline(ctx context.Context, req *devicepb.OfflineRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Offline not implemented")
}
