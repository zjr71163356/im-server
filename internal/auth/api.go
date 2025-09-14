package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"im-server/pkg/dao"
	authpb "im-server/pkg/protocol/pb/authpb"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const AuthKey = "auth:%d"

// AuthDevice 定义了存储在 Redis 中的设备认证信息
type AuthDevice struct {
	DeviceID       uint64 `json:"device_id"`        // 设备ID
	Token          string `json:"token"`            // 设备Token
	TokenExpiresAt int64  `json:"token_expires_at"` // Token过期时间
}

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

// getAuthDevice 从 Redis 获取单个设备的认证信息
func (s *AuthIntService) getAuthDevice(ctx context.Context, userID, deviceID uint64) (*AuthDevice, error) {
	key := fmt.Sprintf(AuthKey, userID)
	bytes, err := s.rdb.HGet(ctx, key, strconv.FormatUint(deviceID, 10)).Bytes()
	if err != nil {
		return nil, err
	}

	var device AuthDevice
	err = json.Unmarshal(bytes, &device)
	return &device, err
}

// Auth 权限校验
func (s *AuthIntService) Auth(ctx context.Context, req *authpb.AuthRequest) (*authpb.AuthResponse, error) {
	// 从Redis获取设备认证信息
	authDevice, err := s.getAuthDevice(ctx, req.UserId, req.DeviceId)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "获取设备认证信息失败: %v", err)
	}

	// 验证token
	if authDevice.Token != req.Token {
		return nil, status.Errorf(codes.Unauthenticated, "token验证失败")
	}

	// 检查token是否过期
	if time.Now().Unix() > authDevice.TokenExpiresAt {
		return nil, status.Errorf(codes.Unauthenticated, "token已过期")
	}

	return &authpb.AuthResponse{
		Valid:   true,
		Message: "token验证成功",
	}, nil
}

// Register 用户注册
func (s *AuthIntService) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	// 检查用户是否存在
	exists, err := s.queries.UserExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, " 检查用户是否存在失败: %v", err)
	}
	if exists {
		// 如果 err 是 nil，说明用户已存在
		return &authpb.RegisterResponse{
			Message: "用户已存在",
			Code:    uint32(codes.InvalidArgument), // 建议使用非零状态码表示失败
		}, status.Errorf(codes.InvalidArgument, "用户已存在: %v", err)
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
		Code:    0, // 建议使用 0 表示成功
	}, nil
}

// Login 用户登录
func (s *AuthIntService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	// 验证用户凭据
	user, err := s.validateUserCredentials(ctx, req.Username, req.Password)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "用户名或密码错误")
	}

	// 生成随机token
	token := generateRandomToken()

	// 设置token过期时间
	expiresAt := time.Now().Add(time.Hour * 24 * 7).Unix() // 7天有效期

	// 将设备认证信息存储到Redis
	authDevice := AuthDevice{
		DeviceID:       req.DeviceId,
		Token:          token,
		TokenExpiresAt: expiresAt,
	}
	err = s.setAuthDevice(ctx, user.ID, req.DeviceId, authDevice)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "存储认证信息失败: %v", err)
	}

	return &authpb.LoginResponse{
		UserId:  user.ID,
		Token:   token,
		Message: "登录成功",
		// Code:    0,
	}, nil
}

// setAuthDevice 将设备认证信息存储到 Redis
func (s *AuthIntService) setAuthDevice(ctx context.Context, userID, deviceID uint64, device AuthDevice) error {
	bytes, err := json.Marshal(device)
	if err != nil {
		return err
	}

	key := fmt.Sprintf(AuthKey, userID)
	_, err = s.rdb.HSet(ctx, key, strconv.FormatUint(deviceID, 10), bytes).Result()
	if err != nil {
		return err
	}
	return nil
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
