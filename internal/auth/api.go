package auth

import (
	"context"
	"errors"
	"im-server/pkg/config"
	"im-server/pkg/dao"
	"im-server/pkg/jwt"
	authpb "im-server/pkg/protocol/pb/authpb"
	"time"

	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthIntService 认证服务
type AuthIntService struct {
	authpb.UnimplementedAuthIntServiceServer
	queries dao.Querier
	rdb     redis.Cmdable
}

// NewAuthIntService 创建一个新的 AuthIntService 实例
func NewAuthIntService(queries dao.Querier, rdb redis.Cmdable) *AuthIntService {
	return &AuthIntService{
		queries: queries,
		rdb:     rdb,
	}
}

// Auth 权限校验：使用 JWT 验证并解析身份
func (s *AuthIntService) Auth(ctx context.Context, req *authpb.AuthRequest) (*authpb.AuthResponse, error) {
	if req.GetToken() == "" {
		return nil, status.Error(codes.Unauthenticated, "empty token")
	}

	// 获取 JWT 配置
	jwtConfig := config.Config.JWT
	secret := []byte(jwtConfig.Secret)

	// 解析并验证 JWT
	_, _, err := jwt.ParseJWT(req.Token, secret, jwtConfig.Issuer, jwtConfig.Audience)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
	}

	return &authpb.AuthResponse{
		Valid:   true,
		Message: "token verified",
		// 注意：需要重新生成 proto 代码后才能返回 user_id/device_id 字段
	}, nil
}

// Register 用户注册
func (s *AuthIntService) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	// 检查用户是否存在
	exists, err := s.queries.UserExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "检查用户是否存在失败: %v", err)
	}
	if exists {
		return &authpb.RegisterResponse{
			Message: "用户已存在",
			Code:    uint32(codes.InvalidArgument),
		}, status.Errorf(codes.InvalidArgument, "用户已存在")
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "密码哈希失败: %v", err)
	}

	// 创建新用户
	result, err := s.queries.CreateUserByUsername(ctx, dao.CreateUserByUsernameParams{
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		Username:       req.Username,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "创建用户失败: %v", err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "获取用户ID失败: %v", err)
	}

	return &authpb.RegisterResponse{
		UserId:  uint64(lastID),
		Message: "注册成功",
		Code:    0,
	}, nil
}

// Login 用户登录
func (s *AuthIntService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	// 验证用户凭据
	user, err := s.validateUserCredentials(ctx, req.Username, req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "用户名或密码错误")
	}

	// 获取 JWT 配置
	jwtConfig := config.Config.JWT
	secret := []byte(jwtConfig.Secret)

	// 解析 TTL
	ttl, err := time.ParseDuration(jwtConfig.TTL)
	if err != nil {
		ttl = 24 * time.Hour // 默认 24 小时
	}

	// 生成 JWT token
	token, err := jwt.GenerateJWT(user.ID, req.DeviceId, secret, ttl, jwtConfig.Issuer, jwtConfig.Audience)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "生成token失败: %v", err)
	}

	expiresAt := time.Now().Add(ttl).Unix()

	return &authpb.LoginResponse{
		UserId:    user.ID,
		Token:     token,
		ExpiresAt: expiresAt,
		Message:   "登录成功",
	}, nil
}

// validateUserCredentials 验证用户凭据
func (s *AuthIntService) validateUserCredentials(ctx context.Context, username, password string) (*dao.GetUserByUsernameForAuthRow, error) {
	user, err := s.queries.GetUserByUsernameForAuth(ctx, username)
	if err != nil {
		return nil, err
	}

	if !verifyPassword(password, user.HashedPassword) {
		return nil, errors.New("invalid password")
	}

	return &user, nil
}
