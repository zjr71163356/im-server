package auth

// filepath: /home/tyrfly/im-server/internal/auth/api_test.go

import (
	"context"
	"database/sql"

	"encoding/json"
	"fmt"
	"im-server/pkg/dao"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mock_dao "im-server/pkg/mocks"
	authpb "im-server/pkg/protocol/pb/authpb"
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
		rdb, _ := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

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
		rdb, _ := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

		req := &authpb.RegisterRequest{Username: "existinguser"}

		// 预期 UserExistsByUsername 会被调用，并返回 true
		queries.EXPECT().
			UserExistsByUsername(gomock.Any(), req.Username).
			Times(1).
			Return(true, nil)

		// --- 调用函数 ---
		res, err := authService.Register(context.Background(), req)

		// --- 断言结果 ---
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "用户已存在", res.Message)
	})
}

func TestAuth(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queries := mock_dao.NewMockQuerier(ctrl)
		rdb, mockRdb := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

		req := &authpb.AuthRequest{
			UserId:   1,
			DeviceId: 100,
			Token:    "valid-token",
		}

		// 准备 mock 的 Redis 返回数据
		expectedDevice := AuthDevice{
			DeviceID:       req.DeviceId,
			Token:          req.Token,
			TokenExpiresAt: time.Now().Add(time.Hour).Unix(),
		}
		expectedBytes, _ := json.Marshal(expectedDevice)
		expectedKey := fmt.Sprintf(AuthKey, req.UserId)
		expectedField := strconv.FormatUint(req.DeviceId, 10)

		// 设置 Redis 的预期行为
		mockRdb.ExpectHGet(expectedKey, expectedField).SetVal(string(expectedBytes))

		// 调用被测试的函数
		_, err := authService.Auth(context.Background(), req)

		// 断言结果
		require.NoError(t, err)
		// 验证所有 Redis 的预期行为都已满足
		require.NoError(t, mockRdb.ExpectationsWereMet())
	})

	t.Run("TokenMismatch", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queries := mock_dao.NewMockQuerier(ctrl)
		rdb, mockRdb := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

		req := &authpb.AuthRequest{
			UserId:   1,
			DeviceId: 100,
			Token:    "invalid-token",
		}

		// Redis 中存储的是 "valid-token"
		expectedDevice := AuthDevice{
			DeviceID:       req.DeviceId,
			Token:          "valid-token",
			TokenExpiresAt: time.Now().Add(time.Hour).Unix(),
		}
		expectedBytes, _ := json.Marshal(expectedDevice)
		expectedKey := fmt.Sprintf(AuthKey, req.UserId)
		expectedField := strconv.FormatUint(req.DeviceId, 10)

		mockRdb.ExpectHGet(expectedKey, expectedField).SetVal(string(expectedBytes))

		_, err := authService.Auth(context.Background(), req)

		require.Error(t, err)
		st, _ := status.FromError(err)
		require.Equal(t, codes.Unauthenticated, st.Code())
		require.Contains(t, st.Message(), "token验证失败")
		require.NoError(t, mockRdb.ExpectationsWereMet())
	})

	t.Run("TokenExpired", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queries := mock_dao.NewMockQuerier(ctrl)
		rdb, mockRdb := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

		req := &authpb.AuthRequest{
			UserId:   1,
			DeviceId: 100,
			Token:    "valid-token",
		}

		// Redis 中存储的是一个已过期的 token
		expectedDevice := AuthDevice{
			DeviceID:       req.DeviceId,
			Token:          "valid-token",
			TokenExpiresAt: time.Now().Add(-time.Hour).Unix(), // 1小时前过期
		}
		expectedBytes, _ := json.Marshal(expectedDevice)
		expectedKey := fmt.Sprintf(AuthKey, req.UserId)
		expectedField := strconv.FormatUint(req.DeviceId, 10)

		mockRdb.ExpectHGet(expectedKey, expectedField).SetVal(string(expectedBytes))

		_, err := authService.Auth(context.Background(), req)

		require.Error(t, err)
		st, _ := status.FromError(err)
		require.Equal(t, codes.Unauthenticated, st.Code())
		require.Contains(t, st.Message(), "token已过期")
		require.NoError(t, mockRdb.ExpectationsWereMet())
	})

	t.Run("RedisError", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queries := mock_dao.NewMockQuerier(ctrl)
		rdb, mockRdb := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

		req := &authpb.AuthRequest{
			UserId:   1,
			DeviceId: 100,
			Token:    "any-token",
		}

		expectedKey := fmt.Sprintf(AuthKey, req.UserId)
		expectedField := strconv.FormatUint(req.DeviceId, 10)

		// 模拟 Redis 返回一个错误
		mockRdb.ExpectHGet(expectedKey, expectedField).SetErr(redis.Nil)

		_, err := authService.Auth(context.Background(), req)

		require.Error(t, err)
		st, _ := status.FromError(err)
		require.Equal(t, codes.Unauthenticated, st.Code())
		require.NoError(t, mockRdb.ExpectationsWereMet())
	})
}

func TestLogin(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queries := mock_dao.NewMockQuerier(ctrl)
		rdb, mockRdb := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

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

		// Mock setAuthDevice call (redis HSet)
		expectedKey := fmt.Sprintf(AuthKey, expectedUser.ID)
		expectedField := strconv.FormatUint(req.DeviceId, 10)

		// We use a custom matcher because the token and expiry are random
		mockRdb.Regexp().ExpectHSet(expectedKey, expectedField, `.*`).SetVal(1)
		res, err := authService.Login(context.Background(), req)

		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "登录成功", res.Message)
		require.Equal(t, expectedUser.ID, res.UserId)
		require.NotEmpty(t, res.Token)
		require.NoError(t, mockRdb.ExpectationsWereMet())
	})

	t.Run("InvalidCredentials", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queries := mock_dao.NewMockQuerier(ctrl)
		rdb, _ := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

		req := &authpb.LoginRequest{
			Username: "testuser",
			Password: "wrongpassword",
			DeviceId: 101,
		}

		// Mock user data from DB with a different password
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
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Unauthenticated, st.Code())
		require.Contains(t, st.Message(), "用户名或密码错误")
	})

	t.Run("UserNotFound", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queries := mock_dao.NewMockQuerier(ctrl)
		rdb, _ := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

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
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Unauthenticated, st.Code())
		require.Contains(t, st.Message(), "用户名或密码错误")
	})

	t.Run("RedisError", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		queries := mock_dao.NewMockQuerier(ctrl)
		rdb, mockRdb := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

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

		// expectedKey := fmt.Sprintf(AuthKey, expectedUser.ID)
		// expectedField := strconv.FormatUint(req.DeviceId, 10)
		// mockRdb.ExpectHSet(expectedKey, expectedField, gomock.Any()).SetErr(errors.New("redis error"))

		_, err = authService.Login(context.Background(), req)

		require.Error(t, err)
		st, ok := status.FromError(err)
		require.True(t, ok)
		require.Equal(t, codes.Internal, st.Code())
		require.Contains(t, st.Message(), "存储认证信息失败")
		require.NoError(t, mockRdb.ExpectationsWereMet())
	})
}
