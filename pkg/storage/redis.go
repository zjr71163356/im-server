package storage

import (
	"log"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func init() {
	InitRedis("localhost:6379", "", 0)
}

// InitRedis 初始化 Redis 客户端
func InitRedis(addr string, password string, db int) {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // 如果没有密码，设置为 ""
		DB:       db,       // 默认数据库
	})

	// 测试连接
	_, err := RedisClient.Ping(RedisClient.Context()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
}
