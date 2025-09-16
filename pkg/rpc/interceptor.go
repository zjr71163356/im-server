package rpc

import (
	"context"
	"strings"

	"im-server/pkg/config"
	"im-server/pkg/jwt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ValidationUnaryInterceptor 返回一个 grpc.UnaryServerInterceptor，
// 在所有 unary RPC 调用中自动调用请求消息的 Validate() 方法（如果存在）。
func ValidationUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if v, ok := req.(interface{ Validate() error }); ok {
			if err := v.Validate(); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
			}
		}
		return handler(ctx, req)
	}
}

// JWTAuthUnaryInterceptor JWT 认证拦截器，验证 token 并注入 user_id 到 context
func JWTAuthUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization header")
		}

		token := authHeaders[0]
		if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = strings.TrimSpace(token[7:])
		}
		if token == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		// 获取 JWT 配置
		jwtConfig := config.Config.JWT
		secret := []byte(jwtConfig.Secret)

		// 解析并验证 JWT
		uid, did, err := jwt.ParseJWT(token, secret, jwtConfig.Issuer, jwtConfig.Audience)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "token verification failed: %v", err)
		}

		// 把 user_id 和 device_id 注入到 ctx，供业务侧读取
		ctx = context.WithValue(ctx, "user_id", uid)
		ctx = context.WithValue(ctx, "device_id", did)
		return handler(ctx, req)
	}
}
