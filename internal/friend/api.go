package friend

import (
	"context"
	"database/sql"
	"im-server/pkg/dao"
	"im-server/pkg/protocol/pb/friendpb"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FriendExtService 好友服务
type FriendExtService struct {
	friendpb.UnimplementedFriendExtServiceServer
	queries dao.Querier
}

// NewFriendExtService 创建一个新的 FriendExtService 实例
func NewFriendExtService(queries dao.Querier) *FriendExtService {
	return &FriendExtService{
		queries: queries,
	}
}

// SendFriendRequest 发送好友申请
func (s *FriendExtService) SendFriendRequest(ctx context.Context, req *friendpb.SendFriendRequestRequest) (*friendpb.SendFriendRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// 从context中获取当前用户ID (这里假设已经通过middleware设置)
	userID, ok := ctx.Value("user_id").(uint64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// 检查是否向自己发送申请
	if userID == req.RecipientId {
		return nil, status.Error(codes.InvalidArgument, "cannot send friend request to yourself")
	}

	// 检查是否已经存在待处理的申请
	existingCount, err := s.queries.CheckExistingRequest(ctx, dao.CheckExistingRequestParams{
		RequesterID: userID,
		RecipientID: req.RecipientId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check existing request")
	}
	if existingCount > 0 {
		return nil, status.Error(codes.AlreadyExists, "friend request already exists")
	}

	// 检查是否已经是好友关系
	friendshipCount, err := s.queries.CheckFriendship(ctx, dao.CheckFriendshipParams{
		UserID:   userID,
		FriendID: req.RecipientId,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to check friendship")
	}
	if friendshipCount > 0 {
		return nil, status.Error(codes.AlreadyExists, "already friends")
	}

	// 创建好友申请
	now := time.Now()
	err = s.queries.CreateFriendRequest(ctx, dao.CreateFriendRequestParams{
		RequesterID: userID,
		RecipientID: req.RecipientId,
		Status:      0, // 0 = 待处理
		Message:     req.Message,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create friend request")
	}

	return &friendpb.SendFriendRequestResponse{
		Message: "Friend request sent successfully",
	}, nil
}

// GetReceivedFriendRequests 获取收到的好友申请列表
func (s *FriendExtService) GetReceivedFriendRequests(ctx context.Context, req *friendpb.GetReceivedFriendRequestsRequest) (*friendpb.GetReceivedFriendRequestsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// 从context中获取当前用户ID
	userID, ok := ctx.Value("user_id").(uint64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// 设置默认分页参数
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	// 获取好友申请列表
	var requests []dao.FriendRequest
	var err error

	if req.Status > 0 {
		// 按状态过滤
		requests, err = s.queries.GetReceivedFriendRequests(ctx, dao.GetReceivedFriendRequestsParams{
			RecipientID: userID,
			Status:      int8(req.Status),
		})
	} else {
		// 获取待处理的申请
		requests, err = s.queries.GetPendingFriendRequests(ctx, userID)
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get friend requests")
	}

	// 转换为protobuf格式
	pbRequests := make([]*friendpb.FriendRequestInfo, len(requests))
	for i, r := range requests {
		pbRequests[i] = &friendpb.FriendRequestInfo{
			Id:          r.ID,
			RequesterId: r.RequesterID,
			RecipientId: r.RecipientID,
			Status:      uint32(r.Status),
			Message:     r.Message,
			CreatedAt:   r.CreatedAt.Unix(),
			UpdatedAt:   r.UpdatedAt.Unix(),
		}
	}

	return &friendpb.GetReceivedFriendRequestsResponse{
		Requests: pbRequests,
		Total:    uint32(len(pbRequests)),
	}, nil
}

// GetSentFriendRequests 获取发送的好友申请列表
func (s *FriendExtService) GetSentFriendRequests(ctx context.Context, req *friendpb.GetSentFriendRequestsRequest) (*friendpb.GetSentFriendRequestsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// 从context中获取当前用户ID
	userID, ok := ctx.Value("user_id").(uint64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// 设置默认分页参数
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	// 获取好友申请列表
	var requests []dao.FriendRequest
	var err error

	if req.Status > 0 {
		// 按状态过滤
		requests, err = s.queries.GetSentFriendRequests(ctx, dao.GetSentFriendRequestsParams{
			RequesterID: userID,
			Status:      int8(req.Status),
		})
	} else {
		// 获取所有发送的申请
		requests, err = s.queries.GetSentFriendRequests(ctx, dao.GetSentFriendRequestsParams{
			RequesterID: userID,
			Status:      0, // 待处理
		})
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get sent friend requests")
	}

	// 转换为protobuf格式
	pbRequests := make([]*friendpb.FriendRequestInfo, len(requests))
	for i, r := range requests {
		pbRequests[i] = &friendpb.FriendRequestInfo{
			Id:          r.ID,
			RequesterId: r.RequesterID,
			RecipientId: r.RecipientID,
			Status:      uint32(r.Status),
			Message:     r.Message,
			CreatedAt:   r.CreatedAt.Unix(),
			UpdatedAt:   r.UpdatedAt.Unix(),
		}
	}

	return &friendpb.GetSentFriendRequestsResponse{
		Requests: pbRequests,
		Total:    uint32(len(pbRequests)),
	}, nil
}

// HandleFriendRequest 处理好友申请（同意/拒绝/忽略）
func (s *FriendExtService) HandleFriendRequest(ctx context.Context, req *friendpb.HandleFriendRequestRequest) (*friendpb.HandleFriendRequestResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// 从context中获取当前用户ID
	userID, ok := ctx.Value("user_id").(uint64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// 获取好友申请详情
	friendRequest, err := s.queries.GetFriendRequest(ctx, req.RequestId)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, status.Error(codes.NotFound, "friend request not found")
		}
		return nil, status.Error(codes.Internal, "failed to get friend request")
	}

	// 验证当前用户是否为申请的接收方
	if friendRequest.RecipientID != userID {
		return nil, status.Error(codes.PermissionDenied, "permission denied")
	}

	// 检查申请状态
	if friendRequest.Status != 0 {
		return nil, status.Error(codes.FailedPrecondition, "friend request already processed")
	}

	now := time.Now()

	// 根据动作更新申请状态
	switch req.Action {
	case 1: // 同意
		err = s.queries.AcceptFriendRequest(ctx, dao.AcceptFriendRequestParams{
			UpdatedAt: now,
			ID:        req.RequestId,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to accept friend request")
		}

		// 创建双向好友关系
		err = s.queries.CreateFriend(ctx, dao.CreateFriendParams{
			UserID:     friendRequest.RecipientID,
			FriendID:   friendRequest.RequesterID,
			Remark:     "",
			CategoryID: 0,
			IsBlocked:  0,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to create friendship")
		}

		err = s.queries.CreateFriend(ctx, dao.CreateFriendParams{
			UserID:     friendRequest.RequesterID,
			FriendID:   friendRequest.RecipientID,
			Remark:     "",
			CategoryID: 0,
			IsBlocked:  0,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to create friendship")
		}

		return &friendpb.HandleFriendRequestResponse{
			Message: "Friend request accepted",
		}, nil

	case 2: // 拒绝
		err = s.queries.RejectFriendRequest(ctx, dao.RejectFriendRequestParams{
			UpdatedAt: now,
			ID:        req.RequestId,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to reject friend request")
		}

		return &friendpb.HandleFriendRequestResponse{
			Message: "Friend request rejected",
		}, nil

	case 3: // 忽略
		err = s.queries.IgnoreFriendRequest(ctx, dao.IgnoreFriendRequestParams{
			UpdatedAt: now,
			ID:        req.RequestId,
		})
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to ignore friend request")
		}

		return &friendpb.HandleFriendRequestResponse{
			Message: "Friend request ignored",
		}, nil

	default:
		return nil, status.Error(codes.InvalidArgument, "invalid action")
	}
}

// GetFriendList 获取好友列表
func (s *FriendExtService) GetFriendList(ctx context.Context, req *friendpb.GetFriendListRequest) (*friendpb.GetFriendListResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// 从context中获取当前用户ID
	userID, ok := ctx.Value("user_id").(uint64)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// 设置默认分页参数
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	// 获取好友列表
	var friends []dao.Friend
	var err error

	if req.CategoryId > 0 {
		// 按分类获取
		friends, err = s.queries.GetUserFriendsByCategory(ctx, dao.GetUserFriendsByCategoryParams{
			UserID:     userID,
			CategoryID: req.CategoryId,
		})
	} else {
		// 获取所有好友
		friends, err = s.queries.GetUserFriends(ctx, userID)
	}

	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get friend list")
	}

	// 转换为protobuf格式
	pbFriends := make([]*friendpb.FriendInfo, len(friends))
	for i, f := range friends {
		pbFriends[i] = &friendpb.FriendInfo{
			UserId:     f.UserID,
			FriendId:   f.FriendID,
			Remark:     f.Remark,
			CategoryId: f.CategoryID,
			IsBlocked:  f.IsBlocked == 1,
			CreatedAt:  f.CreatedAt.Unix(),
			UpdatedAt:  f.UpdatedAt.Unix(),
		}
	}

	return &friendpb.GetFriendListResponse{
		Friends: pbFriends,
		Total:   uint32(len(pbFriends)),
	}, nil
}
