package user

import (
	"im-server/internal/repo"
	"im-server/pkg/protocol/pb/userpb"
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
