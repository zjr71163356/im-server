package user

import (
	"context"
	"database/sql"
	"im-server/pkg/dao"
	"im-server/pkg/protocol/pb/userpb"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserService 用户服务
type UserExtService struct {
	userpb.UnsafeUserExtServiceServer
	queries dao.Querier
}

// NewUserService 创建一个新的 UserService 实例
func NewUserService(queries dao.Querier) *UserExtService {
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

	// 分页参数默认与边界校验
	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := pageSize * (page - 1)

	// 1) 按昵称模糊查询（分页）
	nickNameRows, err := s.queries.ListUsersByNickname(ctx, dao.ListUsersByNicknameParams{
		Nickname: req.Keyword,
		Limit:    int32(pageSize),
		Offset:   int32(offset),
	})

	if err != nil && err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "按昵称搜索用户失败: %v", err)
	}

	if err == nil && nickNameRows != nil && len(nickNameRows) > 0 {
		users := make([]*userpb.UserInfo, 0, len(nickNameRows))
		for _, r := range nickNameRows {
			users = append(users, &userpb.UserInfo{
				UserId:    r.ID,
				Username:  r.Username,
				AvatarUrl: r.AvatarUrl,
			})
		}
		return &userpb.SearchUserResponse{Users: users, Total: uint32(len(users))}, nil
	}

	// 精确按用户名查找（可改为模糊 LIKE）
	nameRow, err := s.queries.GetUserByUsernameForSearch(ctx, req.Keyword)
	if err == nil {
		return &userpb.SearchUserResponse{
			Users: []*userpb.UserInfo{rowToPB(nameRow)},
			Total: 1,
		}, nil
	}
	if err != sql.ErrNoRows {
		return nil, status.Errorf(codes.Internal, "按昵称搜索用户失败: %v", err)
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
