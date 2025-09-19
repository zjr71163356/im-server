package gateway

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"im-server/pkg/config"
	authpb "im-server/pkg/protocol/pb/authpb"
	friendpb "im-server/pkg/protocol/pb/friendpb"
	messagepb "im-server/pkg/protocol/pb/messagepb"
	userpb "im-server/pkg/protocol/pb/userpb"
)

// GatewayServer grpc-gateway 服务器
type GatewayServer struct {
	mux    *runtime.ServeMux
	config *config.Configuration
}

// customErrorHandler 自定义错误处理器
func customErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("grpc-gateway error: %v", err)

	// 获取 gRPC 状态码并转换为 HTTP 状态码
	s := status.Convert(err)
	httpCode := runtime.HTTPStatusFromCode(s.Code())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)

	// 返回统一的错误格式
	errorResponse := map[string]interface{}{
		"code":    int(s.Code()),
		"message": s.Message(),
	}

	if jsonBytes, err := marshaler.Marshal(errorResponse); err == nil {
		w.Write(jsonBytes)
	} else {
		log.Printf("Failed to marshal error response: %v", err)
		w.Write([]byte(`{"code": 13, "message": "Internal error"}`))
	}
}

// NewGatewayServer 创建新的 grpc-gateway 服务器
func NewGatewayServer(cfg *config.Configuration) *GatewayServer {
	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{}),
		runtime.WithErrorHandler(customErrorHandler),
		// 仅透传 Authorization 头到下游 gRPC metadata（仅凭 token 验证）
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			switch strings.ToLower(key) {
			case "authorization":
				return key, true
			default:
				return runtime.DefaultHeaderMatcher(key)
			}
		}),
	)

	return &GatewayServer{
		mux:    mux,
		config: cfg,
	}
}

// RegisterHandlers 注册所有 gRPC 服务的 HTTP 处理器
func (g *GatewayServer) RegisterHandlers() error {
	ctx := context.Background()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// 注册认证服务
	authAddr := g.config.Services.Auth.RPCAddr
	if authAddr == "" {
		authAddr = "localhost:50051" // 默认地址
	}
	if err := authpb.RegisterAuthIntServiceHandlerFromEndpoint(ctx, g.mux, authAddr, opts); err != nil {
		return fmt.Errorf("failed to register auth service at %s: %v", authAddr, err)
	}

	// 注册用户服务
	userAddr := g.config.Services.User.RPCAddr
	if userAddr == "" {
		userAddr = "localhost:50052" // 默认地址
	}
	if err := userpb.RegisterUserExtServiceHandlerFromEndpoint(ctx, g.mux, userAddr, opts); err != nil {
		return fmt.Errorf("failed to register user service at %s: %v", userAddr, err)
	}

	// 注册好友服务
	friendAddr := g.config.Services.Friend.RPCAddr
	if friendAddr == "" {
		friendAddr = "localhost:50053" // 默认地址
	}
	if err := friendpb.RegisterFriendExtServiceHandlerFromEndpoint(ctx, g.mux, friendAddr, opts); err != nil {
		return fmt.Errorf("failed to register friend service at %s: %v", friendAddr, err)
	}

	// 注册消息服务（外部接口）
	messageAddr := g.config.Services.Message.RPCAddr
	if messageAddr == "" {
		messageAddr = "localhost:50056" // 默认地址
	}
	if err := messagepb.RegisterMessageExtServiceHandlerFromEndpoint(ctx, g.mux, messageAddr, opts); err != nil {
		return fmt.Errorf("failed to register message service at %s: %v", messageAddr, err)
	}

	log.Printf("Successfully registered grpc-gateway handlers:")
	log.Printf("  Auth service: %s", authAddr)
	log.Printf("  User service: %s", userAddr)
	log.Printf("  Friend service: %s", friendAddr)
	log.Printf("  Message service: %s", messageAddr)

	return nil
}

// handleHealth 健康检查端点
func (g *GatewayServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok","service":"api-gateway"}`))
}

// ServeHTTP 实现 http.Handler 接口
func (g *GatewayServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 添加 CORS 头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 健康检查端点
	if r.URL.Path == "/health" {
		g.handleHealth(w, r)
		return
	}

	// 记录请求
	log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)

	// 委托给 grpc-gateway
	g.mux.ServeHTTP(w, r)
}
