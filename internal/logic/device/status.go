package device

import (
	"context"
	"fmt"
	"im-server/pkg/storage"

	"time"

	"github.com/go-redis/redis/v8"
)

// SetUserOnline 设置用户为在线状态
func SetUserOnline(userID uint64) error {
	key := fmt.Sprintf("user:online:%d", userID)
	return storage.RedisClient.Set(context.Background(), key, "1", time.Hour).Err()
}

// SetUserOffline 设置用户为离线状态
func SetUserOffline(userID uint64) error {
	key := fmt.Sprintf("user:online:%d", userID)
	return storage.RedisClient.Del(context.Background(), key).Err()
}

// IsUserOnline 检查用户是否在线
func IsUserOnline(userID uint64) (bool, error) {
	key := fmt.Sprintf("user:online:%d", userID)
	result, err := storage.RedisClient.Get(context.Background(), key).Result()
	if err == redis.Nil {
		return false, nil // 用户不在线
	}
	return result == "1", err
}
