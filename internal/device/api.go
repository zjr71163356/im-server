package device

import (
	"context"
	"im-server/pkg/config"
	"im-server/pkg/dao"
	"im-server/pkg/jwt"
	"im-server/pkg/protocol/pb/devicepb"
	"time"

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
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	// 1) 校验 token，确保 user/device 与声明一致
	jwtCfg := config.Config.JWT
	uid, did, err := jwt.ParseJWT(req.Token, []byte(jwtCfg.Secret), jwtCfg.Issuer, jwtCfg.Audience)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}
	if uid != req.UserId || did != req.DeviceId {
		return nil, status.Errorf(codes.PermissionDenied, "token mismatch with request")
	}

	// 2) 校验设备归属
	device, err := s.queries.GetDevice(ctx, req.DeviceId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to get device: %v", err)
	}

	if device.UserID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "device does not belong to user")
	}

	// 3) 更新数据库在线状态
	now := time.Now()
	device.ConnAddr = req.ConnAddr
	device.ClientAddr = req.ClientAddr
	device.Status = OnLine
	device.UpdatedAt = now
	if err := s.queries.UpdateDeviceStatus(ctx, dao.UpdateDeviceStatusParams{
		UpdatedAt:  now,
		Status:     device.Status,
		ConnAddr:   device.ConnAddr,
		ClientAddr: device.ClientAddr,
		ID:         device.ID,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "update device status failed: %v", err)
	}

	// 4) 写入在线状态到 Redis
	if err := SetDeviceOnline(ctx, &device); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set device online: %v", err)
	}

	return new(emptypb.Empty), nil
}

func (s *DeviceIntService) Offline(ctx context.Context, req *devicepb.OfflineRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	device, err := s.queries.GetDevice(ctx, req.DeviceId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "device not found: %v", err)
	}

	// 确保调用方与设备归属一致
	if device.UserID != req.UserId {
		return nil, status.Error(codes.PermissionDenied, "device does not belong to user")
	}

	now := time.Now()
	if err := s.queries.UpdateDeviceOffline(ctx, dao.UpdateDeviceOfflineParams{
		UpdatedAt: now,
		ID:        req.DeviceId,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "update device offline failed: %v", err)
	}

	if err := SetDeviceOffline(ctx, req.DeviceId); err != nil {
		return nil, status.Errorf(codes.Internal, "set device offline failed: %v", err)
	}

	return new(emptypb.Empty), nil
}
