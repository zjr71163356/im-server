package redis

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

var RedisClient *redis.Client

func init() {
	// 允许通过环境变量跳过自动初始化，方便单元测试
	if strings.EqualFold(os.Getenv("IM_REDIS_SKIP_INIT"), "1") {
		log.Printf("skip redis init because IM_REDIS_SKIP_INIT=1")
		return
	}

	addr := os.Getenv("IM_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("IM_REDIS_PASSWORD")
	db := 0
	if dbStr := os.Getenv("IM_REDIS_DB"); dbStr != "" {
		if parsed, err := strconv.Atoi(dbStr); err == nil {
			db = parsed
		}
	}

	if err := InitRedis(addr, password, db); err != nil {
		log.Printf("Failed to connect to Redis (addr=%s): %v", addr, err)
	}
}

// InitRedis 初始化 Redis 客户端
func InitRedis(addr string, password string, db int) error {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // 如果没有密码，设置为 ""
		DB:       db,       // 默认数据库
	})

	// 测试连接
	if err := RedisClient.Ping(context.Background()).Err(); err != nil {
		return err
	}
	return nil
}
