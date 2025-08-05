package device

import (
	"context"
	"im-server/internal/repo"
	"im-server/pkg/protocol/pb/logicpb"

	"google.golang.org/protobuf/types/known/emptypb"
)

type DeviceIntService struct {
	logicpb.UnsafeDeviceIntServiceServer
	queries *repo.Queries
}

func NewDeviceIntService(queries *repo.Queries) *DeviceIntService {
	return &DeviceIntService{queries: queries}
}

func (s *DeviceIntService) ConnSignIn(ctx context.Context, req *logicpb.ConnSignInRequest) (*emptypb.Empty, error) {
	// TODO: 实现登录逻辑
	return new(emptypb.Empty), nil
}
