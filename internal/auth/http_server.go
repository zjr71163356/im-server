package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"im-server/pkg/dao"
	authpb "im-server/pkg/protocol/pb/authpb"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

// HTTPServer HTTP 服务器，包装 gRPC 服务
type HTTPServer struct {
	authService *AuthIntService
}

// NewHTTPServer 创建新的 HTTP 服务器
func NewHTTPServer(queries dao.Querier, rdb redis.Cmdable) *HTTPServer {
	authService := NewAuthIntService(queries, rdb)
	return &HTTPServer{
		authService: authService,
	}
}

// RegisterRoutes 注册 HTTP 路由
func (h *HTTPServer) RegisterRoutes() *mux.Router {
	router := mux.NewRouter()

	// API v1 路由
	api := router.PathPrefix("/api/v1").Subrouter()
	auth := api.PathPrefix("/auth").Subrouter()

	// 注册路由
	auth.HandleFunc("/register", h.handleRegister).Methods("POST")
	auth.HandleFunc("/login", h.handleLogin).Methods("POST")
	auth.HandleFunc("/verify", h.handleAuth).Methods("POST")

	return router
}

// RegisterRequest HTTP 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterResponse HTTP 注册响应
type RegisterResponse struct {
	UserID  uint64 `json:"user_id"`
	Message string `json:"message"`
	Code    int32  `json:"code"`
}

// LoginRequest HTTP 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	DeviceID uint64 `json:"device_id" binding:"required"`
}

// LoginResponse HTTP 登录响应
type LoginResponse struct {
	UserID  uint64 `json:"user_id"`
	Token   string `json:"token"`
	Message string `json:"message"`
}

// AuthRequest HTTP 认证请求
type AuthRequest struct {
	UserID   uint64 `json:"user_id" binding:"required"`
	DeviceID uint64 `json:"device_id" binding:"required"`
	Token    string `json:"token" binding:"required"`
}

// AuthResponse HTTP 认证响应
type AuthResponse struct {
	Message string `json:"message"`
	Valid   bool   `json:"valid"`
}

// handleRegister 处理用户注册
func (h *HTTPServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 调用 gRPC 服务
	grpcReq := &authpb.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
	}

	grpcResp, err := h.authService.Register(context.Background(), grpcReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 转换响应
	resp := RegisterResponse{
		UserID:  grpcResp.UserId,
		Message: grpcResp.Message,
		Code:    grpcResp.Code,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleLogin 处理用户登录
func (h *HTTPServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 调用 gRPC 服务
	grpcReq := &authpb.LoginRequest{
		Username: req.Username,
		Password: req.Password,
		DeviceId: req.DeviceID,
	}

	grpcResp, err := h.authService.Login(context.Background(), grpcReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 转换响应
	resp := LoginResponse{
		UserID:  grpcResp.UserId,
		Token:   grpcResp.Token,
		Message: grpcResp.Message,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleAuth 处理认证验证
func (h *HTTPServer) handleAuth(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// 调用 gRPC 服务
	grpcReq := &authpb.AuthRequest{
		UserId:   req.UserID,
		DeviceId: req.DeviceID,
		Token:    req.Token,
	}

	_, err := h.authService.Auth(context.Background(), grpcReq)

	resp := AuthResponse{
		Valid: err == nil,
	}

	if err != nil {
		resp.Message = err.Error()
	} else {
		resp.Message = "认证成功"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// StartHTTPServer 启动 HTTP 服务器
func StartHTTPServer(port int, queries dao.Querier, rdb redis.Cmdable) error {
	server := NewHTTPServer(queries, rdb)
	router := server.RegisterRoutes()

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("HTTP server starting on %s\n", addr)

	return http.ListenAndServe(addr, router)
}
