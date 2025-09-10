package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"im-server/internal/auth"
	"im-server/internal/user"
	"im-server/pkg/dao"
	authpb "im-server/pkg/protocol/pb/authpb"
	"im-server/pkg/protocol/pb/userpb"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

// GatewayServer API Gateway 服务器
type GatewayServer struct {
	authService *auth.AuthIntService
	// 这里可以添加其他服务的客户端
	userService *user.UserExtService
	// logicService  *logic.LogicIntService
	// connectService *connect.ConnectExtService
}

// NewGatewayServer 创建新的 Gateway 服务器
func NewGatewayServer(queries dao.Querier, rdb redis.Cmdable) *GatewayServer {
	authService := auth.NewAuthIntService(queries, rdb)
	userService := user.NewUserService(queries)

	return &GatewayServer{
		authService: authService,
		userService: userService,
		// 这里可以初始化其他服务的客户端
	}
}

// RegisterRoutes 注册所有服务的 HTTP 路由
func (g *GatewayServer) RegisterRoutes() *mux.Router {
	router := mux.NewRouter()

	// 添加 CORS 中间件
	router.Use(g.corsMiddleware)

	// 添加日志中间件
	router.Use(g.loggingMiddleware)

	// API v1 路由
	api := router.PathPrefix("/api/v1").Subrouter()

	// 注册各个服务的路由
	g.registerAuthRoutes(api)
	g.registerUserRoutes(api)
	g.registerLogicRoutes(api)
	g.registerConnectRoutes(api)

	// 健康检查
	router.HandleFunc("/health", g.handleHealth).Methods("GET")

	return router
}

// registerAuthRoutes 注册认证服务路由
func (g *GatewayServer) registerAuthRoutes(api *mux.Router) {
	auth := api.PathPrefix("/auth").Subrouter()

	auth.HandleFunc("/register", g.handleRegister).Methods("POST")
	auth.HandleFunc("/login", g.handleLogin).Methods("POST")
	auth.HandleFunc("/verify", g.handleAuth).Methods("POST")
}

// registerUserRoutes 注册用户服务路由
func (g *GatewayServer) registerUserRoutes(api *mux.Router) {
	user := api.PathPrefix("/user").Subrouter()

	// 这里可以添加用户相关的路由
	user.HandleFunc("/profile", g.handleGetProfile).Methods("GET")
	user.HandleFunc("/profile", g.handleUpdateProfile).Methods("PUT")
	user.HandleFunc("/search", g.handleSearchUser).Methods("GET")
	// user.HandleFunc("/friends", g.handleGetFriends).Methods("GET")
	// user.HandleFunc("/friends", g.handleAddFriend).Methods("POST")
}

// registerLogicRoutes 注册逻辑服务路由
func (g *GatewayServer) registerLogicRoutes(api *mux.Router) {
	logic := api.PathPrefix("/logic").Subrouter()

	// 这里可以添加逻辑相关的路由
	logic.HandleFunc("/messages", g.handleSendMessage).Methods("POST")
	logic.HandleFunc("/messages", g.handleGetMessages).Methods("GET")
	// logic.HandleFunc("/groups", g.handleCreateGroup).Methods("POST")
	// logic.HandleFunc("/groups/{id}/members", g.handleAddGroupMember).Methods("POST")
}

// registerConnectRoutes 注册连接服务路由
func (g *GatewayServer) registerConnectRoutes(api *mux.Router) {
	connect := api.PathPrefix("/connect").Subrouter()

	// 这里可以添加连接相关的路由
	connect.HandleFunc("/websocket", g.handleWebSocket).Methods("GET")
	// connect.HandleFunc("/status", g.handleConnectionStatus).Methods("GET")
}

// 中间件

// corsMiddleware CORS 中间件
func (g *GatewayServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware 日志中间件
func (g *GatewayServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s %s %s\n", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// 通用响应结构
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// sendJSONResponse 发送 JSON 响应
func (g *GatewayServer) sendJSONResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// sendError 发送错误响应
func (g *GatewayServer) sendError(w http.ResponseWriter, statusCode int, message string) {
	response := APIResponse{
		Code:    statusCode,
		Message: message,
	}
	g.sendJSONResponse(w, statusCode, response)
}

// 健康检查
func (g *GatewayServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := APIResponse{
		Code:    200,
		Message: "Gateway is healthy",
		Data: map[string]string{
			"status":  "ok",
			"service": "api-gateway",
		},
	}
	g.sendJSONResponse(w, http.StatusOK, response)
}

// Auth 服务处理器

// handleRegister 处理用户注册
func (g *GatewayServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// 调用 gRPC 服务
	grpcReq := &authpb.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
	}

	grpcResp, err := g.authService.Register(context.Background(), grpcReq)
	if err != nil {
		g.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := APIResponse{
		Code:    int(grpcResp.Code),
		Message: grpcResp.Message,
		Data: map[string]interface{}{
			"user_id": grpcResp.UserId,
		},
	}

	g.sendJSONResponse(w, http.StatusOK, response)
}

// handleLogin 处理用户登录
func (g *GatewayServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		DeviceID uint64 `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// 调用 gRPC 服务
	grpcReq := &authpb.LoginRequest{
		Username: req.Username,
		Password: req.Password,
		DeviceId: req.DeviceID,
	}

	grpcResp, err := g.authService.Login(context.Background(), grpcReq)
	if err != nil {
		g.sendError(w, http.StatusUnauthorized, err.Error())
		return
	}

	response := APIResponse{
		Code:    200,
		Message: grpcResp.Message,
		Data: map[string]interface{}{
			"user_id": grpcResp.UserId,
			"token":   grpcResp.Token,
		},
	}

	g.sendJSONResponse(w, http.StatusOK, response)
}

// handleAuth 处理认证验证
func (g *GatewayServer) handleAuth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   uint64 `json:"user_id"`
		DeviceID uint64 `json:"device_id"`
		Token    string `json:"token"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// 调用 gRPC 服务
	grpcReq := &authpb.AuthRequest{
		UserId:   req.UserID,
		DeviceId: req.DeviceID,
		Token:    req.Token,
	}

	_, err := g.authService.Auth(context.Background(), grpcReq)

	if err != nil {
		g.sendError(w, http.StatusUnauthorized, err.Error())
		return
	}

	response := APIResponse{
		Code:    200,
		Message: "认证成功",
		Data: map[string]bool{
			"valid": true,
		},
	}

	g.sendJSONResponse(w, http.StatusOK, response)
}

func (g *GatewayServer) handleSearchUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Keyword  string `json:"keyword"`
		Page     int32  `json:"page"`
		PageSize int32  `json:"page_size"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		g.sendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	
	// 调用 gRPC 服务
	grpcReq := &userpb.SearchUserRequest{
		Keyword:  req.Keyword,
		Page:     uint32(req.Page),
		PageSize: uint32(req.PageSize),
	}

	grpcResp, err := g.userService.SearchUser(context.Background(), grpcReq)
	if err != nil {
		g.sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := APIResponse{
		Code:    200,
		Message: "搜索成功",
		Data: map[string]interface{}{
			"users": grpcResp.Users,
		},
	}

	g.sendJSONResponse(w, http.StatusOK, response)
}

// User 服务处理器（占位符）
func (g *GatewayServer) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	g.sendError(w, http.StatusNotImplemented, "User service not implemented yet")
}

func (g *GatewayServer) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	g.sendError(w, http.StatusNotImplemented, "User service not implemented yet")
}

// Logic 服务处理器（占位符）
func (g *GatewayServer) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	g.sendError(w, http.StatusNotImplemented, "Logic service not implemented yet")
}

func (g *GatewayServer) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	g.sendError(w, http.StatusNotImplemented, "Logic service not implemented yet")
}

// Connect 服务处理器（占位符）
func (g *GatewayServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	g.sendError(w, http.StatusNotImplemented, "Connect service not implemented yet")
}
