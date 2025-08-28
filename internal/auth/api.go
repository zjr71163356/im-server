package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"im-server/pkg/dao"
	"im-server/pkg/protocol/pb/authpb"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// AuthIntService 认证服务
type AuthIntService struct {
	authpb.UnimplementedAuthIntServiceServer
	queries *dao.Queries
}

// NewAuthIntService 创建一个新的 AuthIntService 实例
func NewAuthIntService(queries *dao.Queries) *AuthIntService {
	return &AuthIntService{queries: queries}
}

// Auth 权限校验
func (s *AuthIntService) Auth(ctx context.Context, req *authpb.AuthRequest) (*emptypb.Empty, error) {
	// 从Redis获取设备认证信息
	authDevice, err := AuthDeviceGet(req.UserId, req.DeviceId)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "认证失败: %v", err)
	}

	// 验证token
	if authDevice.Token != req.Token {
		return nil, status.Errorf(codes.Unauthenticated, "token验证失败")
	}

	// 检查token是否过期
	if time.Now().Unix() > authDevice.TokenExpiresAt {
		return nil, status.Errorf(codes.Unauthenticated, "token已过期")
	}

	return &emptypb.Empty{}, nil
}

// Login 用户登录
func (s *AuthIntService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	// 验证用户凭据
	user, err := s.validateUserCredentials(ctx, req.Username, req.Password)
	if err != nil {
		return &authpb.LoginResponse{
			Message: err.Error(),
		}, nil
	}

	// 生成token
	token, expiresAt, err := s.generateToken(user.ID, req.DeviceId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "生成token失败: %v", err)
	}

	return &authpb.LoginResponse{
		UserId:    user.ID,
		Token:     token,
		ExpiresAt: expiresAt,
		Message:   "登录成功",
		UserInfo: &authpb.UserInfo{
			Id:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			PhoneNumber: user.PhoneNumber,
			Nickname:    user.Nickname,
			AvatarUrl:   user.AvatarUrl,
		},
	}, nil
}

func (s *AuthIntService) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	// TODO: Implement user registration logic
	_, err := s.queries.GetUserByUsername(ctx, req.Username)
	if err == nil {
		return &authpb.RegisterResponse{
			Message: "用户已存在",
		}, nil
	}
	// 如果错误不是 "未找到记录"，则是一个真正的数据库错误
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, status.Errorf(codes.Internal, "查询用户失败: %v", err)
	}
	// 在实际应用中应使用更安全的随机盐值
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "注册失败: %v", err)
	}
	// 创建新用户
	result, err := s.queries.CreateUserByUsername(ctx, dao.CreateUserByUsernameParams{
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Username:       req.Username,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "注册失败: %v", err)
	}
	lastID, err := result.LastInsertId()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "获取用户ID失败: %v", err)
	}

	return &authpb.RegisterResponse{
		UserId:  uint64(lastID),
		Message: "注册成功",
	}, nil
}

// validateUserCredentials 验证用户凭据
func (s *AuthIntService) validateUserCredentials(ctx context.Context, username, password string) (*dao.User, error) {
	var user dao.User

	// 通过用户名获取认证信息
	authRow, err := s.queries.GetUserByUsernameForAuth(ctx, username)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	if !verifyPassword(password, authRow.HashedPassword) {
		return nil, errors.New("密码错误")
	}

	user, err = s.queries.GetUser(ctx, authRow.ID)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// generateToken 生成访问token
func (s *AuthIntService) generateToken(userID, deviceID uint64) (string, int64, error) {
	// 生成随机token
	token := generateRandomToken()
	expiresAt := time.Now().Add(30 * 24 * time.Hour).Unix() // 30天过期

	// 存储到Redis
	authDevice := AuthDevice{
		DeviceID:       deviceID,
		Token:          token,
		TokenExpiresAt: expiresAt,
	}

	err := AuthDeviceSet(userID, deviceID, authDevice)
	if err != nil {
		return "", 0, err
	}

	return token, expiresAt, nil
}
