package user

import (
	"context"
	"im-server/internal/repo"
	"im-server/pkg/auth"
	"im-server/pkg/protocol/pb/userpb"

	"google.golang.org/protobuf/types/known/emptypb"
)

// UserIntService 模拟用户服务
type UserIntService struct {
	userpb.UnsafeUserIntServiceServer // 继承 gRPC 的 UserIntServiceServer
	queries                           *repo.Queries
}

// NewUserIntService 创建一个新的 UserIntService 实例
func NewUserIntService(queries *repo.Queries) *UserIntService {
	return &UserIntService{queries: queries}
}

func (s *UserIntService) Auth(ctx context.Context, req *userpb.AuthRequest) (*emptypb.Empty, error) {
	auth.AuthRepo.Get(req.UserId, req.DeviceId)
	// 进行用户认证逻辑
	return &emptypb.Empty{}, nil
}
