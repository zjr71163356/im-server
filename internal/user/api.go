package user

import (
	"context"
	"im-server/pkg/dao"
	"im-server/pkg/protocol/pb/userpb"

	"google.golang.org/protobuf/types/known/emptypb"
)

// UserIntService 模拟用户服务
type UserIntService struct {
	userpb.UnsafeUserIntServiceServer // 继承 gRPC 的 UserIntServiceServer
	queries                           *dao.Queries
}

// NewUserIntService 创建一个新的 UserIntService 实例
func NewUserIntService(queries *dao.Queries) *UserIntService {
	return &UserIntService{queries: queries}
}

func (s *UserIntService) Auth(ctx context.Context, req *userpb.AuthRequest) (*emptypb.Empty, error) {

	err := Auth(ctx, req.UserId, req.DeviceId, req.Token)
	if err != nil {
		return nil, err // 返回认证错误
	}
	// 进行用户认证逻辑
	return &emptypb.Empty{}, nil
}

