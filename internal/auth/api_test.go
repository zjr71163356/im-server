package auth

// filepath: /home/tyrfly/im-server/internal/auth/api_test.go

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"im-server/pkg/config"
	"im-server/pkg/dao"
	"im-server/pkg/jwt"
	mock_dao "im-server/pkg/mocks"
	authpb "im-server/pkg/protocol/pb/authpb"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

// mockResult 是一个 sql.Result 的简单模拟实现
type mockResult int

func (m *mockResult) LastInsertId() (int64, error) {
	return 1, nil // 模拟新用户的 ID 为 1
}

func (m *mockResult) RowsAffected() (int64, error) {
	return 1, nil
}

func TestRegister(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queries := mock_dao.NewMockQuerier(ctrl)
		authService := NewAuthIntService(queries, nil)

		req := &authpb.RegisterRequest{
			Username: "testuser",
			Password: "password",
		}

		queries.EXPECT().
			UserExistsByUsername(gomock.Any(), req.Username).
			Times(1).
			Return(false, nil)

		queries.EXPECT().
			CreateUserByUsername(gomock.Any(), gomock.Any()).
			Times(1).
			Return(new(mockResult), nil)

		res, err := authService.Register(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "注册成功", res.Message)
		require.Equal(t, uint64(1), res.UserId)
	})

	t.Run("UserAlreadyExists", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queries := mock_dao.NewMockQuerier(ctrl)
		authService := NewAuthIntService(queries, nil)

		req := &authpb.RegisterRequest{Username: "existinguser"}

		queries.EXPECT().
			UserExistsByUsername(gomock.Any(), req.Username).
			Times(1).
			Return(true, nil)

		res, err := authService.Register(context.Background(), req)

		require.Error(t, err)
		require.NotNil(t, res)
		require.Equal(t, "用户已存在", res.Message)
	})
}

func TestAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queries := mock_dao.NewMockQuerier(ctrl)
	authService := NewAuthIntService(queries, nil)
	jwtCfg := config.Config.JWT
	secret := []byte(jwtCfg.Secret)

	t.Run("Success", func(t *testing.T) {
		// 生成有效 token
		token, err := jwt.GenerateJWT(1, 100, secret, time.Hour, jwtCfg.Issuer, jwtCfg.Audience)
		require.NoError(t, err)

		_, err = authService.Auth(context.Background(), &authpb.AuthRequest{Token: token})
		require.NoError(t, err)
	})

	t.Run("InvalidToken", func(t *testing.T) {
		// 使用损坏的 token
		invalid := "tampered." + "token"
		_, err := authService.Auth(context.Background(), &authpb.AuthRequest{Token: invalid})
		require.Error(t, err)
	})

	t.Run("ExpiredToken", func(t *testing.T) {
		// 生成已过期 token（负 ttl）
		token, err := jwt.GenerateJWT(1, 100, secret, -time.Hour, jwtCfg.Issuer, jwtCfg.Audience)
		require.NoError(t, err)

		_, err = authService.Auth(context.Background(), &authpb.AuthRequest{Token: token})
		require.Error(t, err)
	})
}

func TestLogin(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queries := mock_dao.NewMockQuerier(ctrl)
	authService := NewAuthIntService(queries, nil)
	jwtCfg := config.Config.JWT
	secret := []byte(jwtCfg.Secret)

	t.Run("Success", func(t *testing.T) {
		req := &authpb.LoginRequest{
			Username: "testuser",
			Password: "password",
			DeviceId: 101,
		}

		// Mock user data from DB
		hashedPassword, err := hashPassword(req.Password)
		require.NoError(t, err)
		expectedUser := dao.GetUserByUsernameForAuthRow{
			ID:             1,
			HashedPassword: hashedPassword,
		}

		// Mock validateUserCredentials call
		queries.EXPECT().
			GetUserByUsernameForAuth(gomock.Any(), req.Username).
			Times(1).
			Return(expectedUser, nil)

		res, err := authService.Login(context.Background(), req)
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, expectedUser.ID, res.UserId)
		require.NotEmpty(t, res.Token)

		// 解析 token 验证 payload
		uid, did, err := jwt.ParseJWT(res.Token, secret, jwtCfg.Issuer, jwtCfg.Audience)
		require.NoError(t, err)
		require.Equal(t, expectedUser.ID, uid)
		require.Equal(t, req.DeviceId, did)
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		req := &authpb.LoginRequest{
			Username: "testuser",
			Password: "wrongpassword",
			DeviceId: 101,
		}

		hashedPassword, err := hashPassword("correctpassword")
		require.NoError(t, err)
		expectedUser := dao.GetUserByUsernameForAuthRow{
			ID:             1,
			HashedPassword: hashedPassword,
		}

		queries.EXPECT().
			GetUserByUsernameForAuth(gomock.Any(), req.Username).
			Times(1).
			Return(expectedUser, nil)

		_, err = authService.Login(context.Background(), req)
		require.Error(t, err)
	})

	t.Run("UserNotFound", func(t *testing.T) {
		req := &authpb.LoginRequest{
			Username: "nonexistentuser",
			Password: "password",
			DeviceId: 101,
		}

		queries.EXPECT().
			GetUserByUsernameForAuth(gomock.Any(), req.Username).
			Times(1).
			Return(dao.GetUserByUsernameForAuthRow{}, sql.ErrNoRows)

		_, err := authService.Login(context.Background(), req)
		require.Error(t, err)
	})

	t.Run("StorageError", func(t *testing.T) {
		// 登录成功但模拟存储错误：当前实现不依赖 redis 存储，因此此场景检查生成 token 是否成功
		req := &authpb.LoginRequest{
			Username: "testuser",
			Password: "password",
			DeviceId: 101,
		}

		hashedPassword, err := hashPassword(req.Password)
		require.NoError(t, err)
		expectedUser := dao.GetUserByUsernameForAuthRow{
			ID:             1,
			HashedPassword: hashedPassword,
		}

		queries.EXPECT().
			GetUserByUsernameForAuth(gomock.Any(), req.Username).
			Times(1).
			Return(expectedUser, nil)

		res, err := authService.Login(context.Background(), req)
		require.NoError(t, err)
		require.NotEmpty(t, res.Token)
		// 解析 token 确认有效
		_, _, err = jwt.ParseJWT(res.Token, secret, jwtCfg.Issuer, jwtCfg.Audience)
		require.NoError(t, err)
	})
}
