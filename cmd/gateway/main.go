package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"

	"im-server/gateway"
	"im-server/pkg/config"
	"im-server/pkg/dao"
)

func main() {
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to database using DSN
	db, err := sql.Open("mysql", cfg.Database.MySQL.DSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	queries := dao.New(db)

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Database.Redis.Host,
		Password: cfg.Database.Redis.Password,
		DB:       0, // use default DB
	})

	gatewayServer := gateway.NewGatewayServer(queries, rdb)
	router := gatewayServer.RegisterRoutes()

	port := cfg.Services.Gateway.Port
	if port == 0 {
		port = 8080
	}

	log.Printf("Starting gateway server on port %d", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), router))
}
