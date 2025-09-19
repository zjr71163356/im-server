package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"im-server/gateway"
	"im-server/pkg/config"
)

func main() {
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建 grpc-gateway 服务器
	gatewayServer := gateway.NewGatewayServer(cfg)

	// 注册所有服务的 HTTP 处理器
	if err := gatewayServer.RegisterHandlers(); err != nil {
		log.Fatalf("Failed to register handlers: %v", err)
	}

	port := cfg.Services.Gateway.Port
	if port == 0 {
		port = 8080
	}

	log.Printf("Starting grpc-gateway server on port %d", port)
	log.Printf("Registered endpoints:")
	log.Printf("  POST /api/v1/auth/register - User registration")
	log.Printf("  POST /api/v1/auth/login - User login")
	log.Printf("  POST /api/v1/auth/verify - Token verification")
	log.Printf("  POST /api/v1/user/search - User search")
	log.Printf("  POST /api/v1/message - Send message")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), gatewayServer))
}
