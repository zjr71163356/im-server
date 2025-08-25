package user

import (
	"im-server/pkg/dao"
)

// UserService 用户服务
type UserService struct {
	queries *dao.Queries
}

// NewUserService 创建一个新的 UserService 实例
func NewUserService(queries *dao.Queries) *UserService {
	return &UserService{queries: queries}
}
