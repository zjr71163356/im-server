package user

import (
	"context"
	"database/sql"
	"im-server/pkg/dao"
	"im-server/pkg/protocol/pb/userpb"
	"strings"
)

// UserService 用户服务
type UserExtService struct {
	userpb.UnsafeUserSearchServiceServer
	queries *dao.Queries
}

// NewUserService 创建一个新的 UserService 实例
func NewUserService(queries *dao.Queries) *UserExtService {
	return &UserExtService{queries: queries}
}
func rowToPB(r interface{}) *userpb.UserInfo {
	switch v := r.(type) {
	case dao.GetUserByUsernameForSearchRow:
		return &userpb.UserInfo{
			UserId:    v.ID,
			Username:  v.Username,
			AvatarUrl: v.AvatarUrl,
		}
	case dao.GetUserByPhoneRow:
		return &userpb.UserInfo{
			UserId:    v.ID,
			Username:  v.Username,
			AvatarUrl: v.AvatarUrl,
		}
	default:
		return &userpb.UserInfo{}
	}
}

func (s *UserExtService) SearchUser(ctx context.Context, req *userpb.SearchUserRequest) (*userpb.SearchUserResponse, error) {
	if req == nil || strings.TrimSpace(req.Keyword) == "" {
		return &userpb.SearchUserResponse{
			Users: []*userpb.UserInfo{},
			Total: 0,
		}, nil
	}
	req.Keyword = strings.TrimSpace(req.Keyword)

	s.queries.ListUsersByNickname(ctx, dao.ListUsersByNicknameParams{
		Nickname: req.Keyword,
		Limit:    int32(req.PageSize),
		Offset:   int32(req.PageSize * (req.Page - 1)),
	})
	// 精确按用户名查找（可改为模糊 LIKE）
	nameRow, err := s.queries.GetUserByUsernameForSearch(ctx, req.Keyword)
	if err == nil {
		return &userpb.SearchUserResponse{
			Users: []*userpb.UserInfo{rowToPB(nameRow)},
			Total: 1,
		}, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// 回退按手机号查找
	phoneRow, err := s.queries.GetUserByPhone(ctx, sql.NullString{String: req.Keyword, Valid: true})
	if err == nil {
		return &userpb.SearchUserResponse{
			Users: []*userpb.UserInfo{rowToPB(phoneRow)},
			Total: 1,
		}, nil
	}
	if err == sql.ErrNoRows {
		// 没有命中，返回空结果而不是错误
		return &userpb.SearchUserResponse{
			Users: []*userpb.UserInfo{},
			Total: 0,
		}, nil
	}

	return nil, err
}
