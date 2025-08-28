package auth

// filepath: /home/tyrfly/im-server/internal/auth/api_test.go

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"im-server/pkg/dao"
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
			GetUserByUsername(gomock.Any(), req.Username).
			Times(1).
			Return(dao.User{}, sql.ErrNoRows)

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
		rdb, mockRdb := redismock.NewClientMock()
		authService := NewAuthIntService(queries, rdb)

		req := &authpb.RegisterRequest{Username: "existinguser"}

		// 预期 GetUserByUsername 会被调用，并返回一个存在的用户
		queries.EXPECT().
			GetUserByUsername(gomock.Any(), req.Username).
			Times(1).
			Return(dao.User{ID: 2, Username: "existinguser"}, nil)

		// --- 调用函数 ---
		res, err := authService.Register(context.Background(), req)

		// --- 断言结果 ---
		require.NoError(t, err)
		require.NotNil(t, res)
		require.Equal(t, "用户已存在", res.Message)

		// 验证 Redis 的预期
		require.NoError(t, mockRdb.ExpectationsWereMet())
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
