package friend

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"im-server/pkg/dao"
	mock_dao "im-server/pkg/mocks"
	"im-server/pkg/protocol/pb/friendpb"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 测试SendFriendRequest接口
func TestSendFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queries := mock_dao.NewMockQuerier(ctrl)
	service := NewFriendExtService(queries)

	t.Run("成功发送好友申请", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.SendFriendRequestRequest{
			RecipientId: 2,
			Message:     "你好，加个好友吧",
		}

		// 模拟查询：不存在待处理的申请
		queries.EXPECT().
			CheckExistingRequest(gomock.Any(), dao.CheckExistingRequestParams{
				RequesterID: uint64(1),
				RecipientID: uint64(2),
			}).
			Return(int64(0), nil)

		// 模拟查询：不是好友关系
		queries.EXPECT().
			CheckFriendship(gomock.Any(), dao.CheckFriendshipParams{
				UserID:   uint64(1),
				FriendID: uint64(2),
			}).
			Return(int64(0), nil)

		// 模拟创建好友申请
		queries.EXPECT().
			CreateFriendRequest(gomock.Any(), gomock.Any()).
			Return(nil)

		resp, err := service.SendFriendRequest(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Message, "successfully")
	})

	t.Run("向自己发送好友申请应该失败", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.SendFriendRequestRequest{
			RecipientId: 1, // 和当前用户ID相同
			Message:     "自己加自己",
		}

		resp, err := service.SendFriendRequest(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st := status.Convert(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "yourself")
	})

	t.Run("重复发送好友申请应该失败", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.SendFriendRequestRequest{
			RecipientId: 2,
			Message:     "重复申请",
		}

		// 模拟查询：存在待处理的申请
		queries.EXPECT().
			CheckExistingRequest(gomock.Any(), dao.CheckExistingRequestParams{
				RequesterID: uint64(1),
				RecipientID: uint64(2),
			}).
			Return(int64(1), nil)

		resp, err := service.SendFriendRequest(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st := status.Convert(err)
		assert.Equal(t, codes.AlreadyExists, st.Code())
		assert.Contains(t, st.Message(), "already exists")
	})

	t.Run("已经是好友关系应该失败", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.SendFriendRequestRequest{
			RecipientId: 2,
			Message:     "已经是好友了",
		}

		// 模拟查询：不存在待处理的申请
		queries.EXPECT().
			CheckExistingRequest(gomock.Any(), gomock.Any()).
			Return(int64(0), nil)

		// 模拟查询：已经是好友关系
		queries.EXPECT().
			CheckFriendship(gomock.Any(), dao.CheckFriendshipParams{
				UserID:   uint64(1),
				FriendID: uint64(2),
			}).
			Return(int64(1), nil)

		resp, err := service.SendFriendRequest(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st := status.Convert(err)
		assert.Equal(t, codes.AlreadyExists, st.Code())
		assert.Contains(t, st.Message(), "already friends")
	})

	t.Run("未认证用户应该失败", func(t *testing.T) {
		ctx := context.Background() // 没有设置user_id
		req := &friendpb.SendFriendRequestRequest{
			RecipientId: 2,
			Message:     "未认证",
		}

		resp, err := service.SendFriendRequest(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st := status.Convert(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("空请求应该失败", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))

		resp, err := service.SendFriendRequest(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st := status.Convert(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}

// 测试GetReceivedFriendRequests接口
func TestGetReceivedFriendRequests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queries := mock_dao.NewMockQuerier(ctrl)
	service := NewFriendExtService(queries)

	t.Run("成功获取收到的好友申请", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.GetReceivedFriendRequestsRequest{
			Page:     1,
			PageSize: 10,
		}

		now := time.Now()
		mockRequests := []dao.FriendRequest{
			{
				ID:          1,
				RequesterID: 2,
				RecipientID: 1,
				Status:      0,
				Message:     "测试申请1",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
			{
				ID:          2,
				RequesterID: 3,
				RecipientID: 1,
				Status:      0,
				Message:     "测试申请2",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		}

		// 模拟获取待处理的申请
		queries.EXPECT().
			GetPendingFriendRequests(gomock.Any(), uint64(1)).
			Return(mockRequests, nil)

		resp, err := service.GetReceivedFriendRequests(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Requests, 2)
		assert.Equal(t, uint32(2), resp.Total)
		assert.Equal(t, uint64(1), resp.Requests[0].Id)
		assert.Equal(t, uint64(2), resp.Requests[0].RequesterId)
		assert.Equal(t, "测试申请1", resp.Requests[0].Message)
	})

	t.Run("按状态过滤好友申请", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.GetReceivedFriendRequestsRequest{
			Page:     1,
			PageSize: 10,
			Status:   1, // 已同意
		}

		now := time.Now()
		mockRequests := []dao.FriendRequest{
			{
				ID:          1,
				RequesterID: 2,
				RecipientID: 1,
				Status:      1,
				Message:     "已同意的申请",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		}

		// 模拟按状态获取申请
		queries.EXPECT().
			GetReceivedFriendRequests(gomock.Any(), dao.GetReceivedFriendRequestsParams{
				RecipientID: uint64(1),
				Status:      int8(1),
			}).
			Return(mockRequests, nil)

		resp, err := service.GetReceivedFriendRequests(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Requests, 1)
		assert.Equal(t, uint32(1), resp.Requests[0].Status)
	})

	t.Run("默认分页参数", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.GetReceivedFriendRequestsRequest{
			// 不设置分页参数，测试默认值
		}

		queries.EXPECT().
			GetPendingFriendRequests(gomock.Any(), uint64(1)).
			Return([]dao.FriendRequest{}, nil)

		resp, err := service.GetReceivedFriendRequests(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

// 测试HandleFriendRequest接口
func TestHandleFriendRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queries := mock_dao.NewMockQuerier(ctrl)
	service := NewFriendExtService(queries)

	now := time.Now()
	mockRequest := dao.FriendRequest{
		ID:          1,
		RequesterID: 2,
		RecipientID: 1,
		Status:      0, // 待处理
		Message:     "请求加好友",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	t.Run("同意好友申请成功", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.HandleFriendRequestRequest{
			RequestId: 1,
			Action:    1, // 同意
		}

		// 模拟获取好友申请
		queries.EXPECT().
			GetFriendRequest(gomock.Any(), uint64(1)).
			Return(mockRequest, nil)

		// 模拟接受申请
		queries.EXPECT().
			AcceptFriendRequest(gomock.Any(), gomock.Any()).
			Return(nil)

		// 模拟创建双向好友关系
		queries.EXPECT().
			CreateFriend(gomock.Any(), gomock.Any()).
			Return(nil).
			Times(2)

		resp, err := service.HandleFriendRequest(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Message, "accepted")
	})

	t.Run("拒绝好友申请成功", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.HandleFriendRequestRequest{
			RequestId: 1,
			Action:    2, // 拒绝
		}

		queries.EXPECT().
			GetFriendRequest(gomock.Any(), uint64(1)).
			Return(mockRequest, nil)

		queries.EXPECT().
			RejectFriendRequest(gomock.Any(), gomock.Any()).
			Return(nil)

		resp, err := service.HandleFriendRequest(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Message, "rejected")
	})

	t.Run("忽略好友申请成功", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.HandleFriendRequestRequest{
			RequestId: 1,
			Action:    3, // 忽略
		}

		queries.EXPECT().
			GetFriendRequest(gomock.Any(), uint64(1)).
			Return(mockRequest, nil)

		queries.EXPECT().
			IgnoreFriendRequest(gomock.Any(), gomock.Any()).
			Return(nil)

		resp, err := service.HandleFriendRequest(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Contains(t, resp.Message, "ignored")
	})

	t.Run("申请不存在应该失败", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.HandleFriendRequestRequest{
			RequestId: 999,
			Action:    1,
		}

		queries.EXPECT().
			GetFriendRequest(gomock.Any(), uint64(999)).
			Return(dao.FriendRequest{}, sql.ErrNoRows)

		resp, err := service.HandleFriendRequest(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st := status.Convert(err)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	t.Run("非接收方不能处理申请", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(3)) // 不是接收方
		req := &friendpb.HandleFriendRequestRequest{
			RequestId: 1,
			Action:    1,
		}

		queries.EXPECT().
			GetFriendRequest(gomock.Any(), uint64(1)).
			Return(mockRequest, nil)

		resp, err := service.HandleFriendRequest(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st := status.Convert(err)
		assert.Equal(t, codes.PermissionDenied, st.Code())
	})

	t.Run("已处理的申请不能再次处理", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.HandleFriendRequestRequest{
			RequestId: 1,
			Action:    1,
		}

		processedRequest := mockRequest
		processedRequest.Status = 1 // 已同意

		queries.EXPECT().
			GetFriendRequest(gomock.Any(), uint64(1)).
			Return(processedRequest, nil)

		resp, err := service.HandleFriendRequest(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st := status.Convert(err)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
		assert.Contains(t, st.Message(), "already processed")
	})

	t.Run("无效的动作应该失败", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.HandleFriendRequestRequest{
			RequestId: 1,
			Action:    999, // 无效动作
		}

		queries.EXPECT().
			GetFriendRequest(gomock.Any(), uint64(1)).
			Return(mockRequest, nil)

		resp, err := service.HandleFriendRequest(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st := status.Convert(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Contains(t, st.Message(), "invalid action")
	})
}

// 测试GetFriendList接口
func TestGetFriendList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queries := mock_dao.NewMockQuerier(ctrl)
	service := NewFriendExtService(queries)

	t.Run("成功获取好友列表", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.GetFriendListRequest{
			Page:     1,
			PageSize: 10,
		}

		now := time.Now()
		mockFriends := []dao.Friend{
			{
				ID:         1,
				UserID:     1,
				FriendID:   2,
				Remark:     "好友1",
				CategoryID: 0,
				IsBlocked:  0,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
			{
				ID:         2,
				UserID:     1,
				FriendID:   3,
				Remark:     "好友2",
				CategoryID: 1,
				IsBlocked:  0,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		}

		queries.EXPECT().
			GetUserFriends(gomock.Any(), uint64(1)).
			Return(mockFriends, nil)

		resp, err := service.GetFriendList(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Friends, 2)
		assert.Equal(t, uint32(2), resp.Total)
		assert.Equal(t, uint64(2), resp.Friends[0].FriendId)
		assert.Equal(t, "好友1", resp.Friends[0].Remark)
		assert.False(t, resp.Friends[0].IsBlocked)
	})

	t.Run("按分类获取好友列表", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.GetFriendListRequest{
			Page:       1,
			PageSize:   10,
			CategoryId: 1,
		}

		now := time.Now()
		mockFriends := []dao.Friend{
			{
				ID:         1,
				UserID:     1,
				FriendID:   3,
				Remark:     "分类1的好友",
				CategoryID: 1,
				IsBlocked:  0,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		}

		queries.EXPECT().
			GetUserFriendsByCategory(gomock.Any(), dao.GetUserFriendsByCategoryParams{
				UserID:     uint64(1),
				CategoryID: uint64(1),
			}).
			Return(mockFriends, nil)

		resp, err := service.GetFriendList(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Friends, 1)
		assert.Equal(t, uint64(1), resp.Friends[0].CategoryId)
	})

	t.Run("空好友列表", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.GetFriendListRequest{
			Page:     1,
			PageSize: 10,
		}

		queries.EXPECT().
			GetUserFriends(gomock.Any(), uint64(1)).
			Return([]dao.Friend{}, nil)

		resp, err := service.GetFriendList(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Friends, 0)
		assert.Equal(t, uint32(0), resp.Total)
	})

	t.Run("默认分页参数", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.GetFriendListRequest{
			// 不设置分页参数
		}

		queries.EXPECT().
			GetUserFriends(gomock.Any(), uint64(1)).
			Return([]dao.Friend{}, nil)

		resp, err := service.GetFriendList(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

// 测试GetSentFriendRequests接口
func TestGetSentFriendRequests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queries := mock_dao.NewMockQuerier(ctrl)
	service := NewFriendExtService(queries)

	t.Run("成功获取发送的好友申请", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.GetSentFriendRequestsRequest{
			Page:     1,
			PageSize: 10,
		}

		now := time.Now()
		mockRequests := []dao.FriendRequest{
			{
				ID:          1,
				RequesterID: 1,
				RecipientID: 2,
				Status:      0,
				Message:     "发送的申请1",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		}

		queries.EXPECT().
			GetSentFriendRequests(gomock.Any(), dao.GetSentFriendRequestsParams{
				RequesterID: uint64(1),
				Status:      int8(0),
			}).
			Return(mockRequests, nil)

		resp, err := service.GetSentFriendRequests(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Requests, 1)
		assert.Equal(t, uint32(1), resp.Total)
		assert.Equal(t, uint64(1), resp.Requests[0].RequesterId)
		assert.Equal(t, uint64(2), resp.Requests[0].RecipientId)
	})

	t.Run("按状态过滤发送的申请", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), "user_id", uint64(1))
		req := &friendpb.GetSentFriendRequestsRequest{
			Page:     1,
			PageSize: 10,
			Status:   1, // 已同意
		}

		now := time.Now()
		mockRequests := []dao.FriendRequest{
			{
				ID:          1,
				RequesterID: 1,
				RecipientID: 2,
				Status:      1,
				Message:     "已被同意的申请",
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		}

		queries.EXPECT().
			GetSentFriendRequests(gomock.Any(), dao.GetSentFriendRequestsParams{
				RequesterID: uint64(1),
				Status:      int8(1),
			}).
			Return(mockRequests, nil)

		resp, err := service.GetSentFriendRequests(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Requests, 1)
		assert.Equal(t, uint32(1), resp.Requests[0].Status)
	})
}

// 通用测试：所有接口的认证检查
func TestAuthenticationRequired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queries := mock_dao.NewMockQuerier(ctrl)
	service := NewFriendExtService(queries)

	ctxWithoutAuth := context.Background()

	t.Run("SendFriendRequest需要认证", func(t *testing.T) {
		req := &friendpb.SendFriendRequestRequest{RecipientId: 2}
		_, err := service.SendFriendRequest(ctxWithoutAuth, req)
		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("GetReceivedFriendRequests需要认证", func(t *testing.T) {
		req := &friendpb.GetReceivedFriendRequestsRequest{}
		_, err := service.GetReceivedFriendRequests(ctxWithoutAuth, req)
		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("GetSentFriendRequests需要认证", func(t *testing.T) {
		req := &friendpb.GetSentFriendRequestsRequest{}
		_, err := service.GetSentFriendRequests(ctxWithoutAuth, req)
		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("HandleFriendRequest需要认证", func(t *testing.T) {
		req := &friendpb.HandleFriendRequestRequest{RequestId: 1, Action: 1}
		_, err := service.HandleFriendRequest(ctxWithoutAuth, req)
		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("GetFriendList需要认证", func(t *testing.T) {
		req := &friendpb.GetFriendListRequest{}
		_, err := service.GetFriendList(ctxWithoutAuth, req)
		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})
}

// 通用测试：空请求处理
func TestNilRequestHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	queries := mock_dao.NewMockQuerier(ctrl)
	service := NewFriendExtService(queries)

	ctx := context.WithValue(context.Background(), "user_id", uint64(1))

	t.Run("SendFriendRequest空请求", func(t *testing.T) {
		_, err := service.SendFriendRequest(ctx, nil)
		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("GetReceivedFriendRequests空请求", func(t *testing.T) {
		_, err := service.GetReceivedFriendRequests(ctx, nil)
		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("GetSentFriendRequests空请求", func(t *testing.T) {
		_, err := service.GetSentFriendRequests(ctx, nil)
		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("HandleFriendRequest空请求", func(t *testing.T) {
		_, err := service.HandleFriendRequest(ctx, nil)
		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("GetFriendList空请求", func(t *testing.T) {
		_, err := service.GetFriendList(ctx, nil)
		assert.Error(t, err)
		st := status.Convert(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}
