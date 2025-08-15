package device

import (
	"context"
	"im-server/internal/repo"
	redisPkg "im-server/pkg/redis"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	deviceInfoKey = "device:info:"
	OnLine        = 1 // 设备在线
	OffLine       = 0 // 设备离线
)

// SetDeviceOnline 设置设备在线信息到redis
func SetDeviceOnline(ctx context.Context, device *repo.Device) error {
	key := deviceInfoKey + strconv.FormatUint(device.ID, 10)
	device.Status = OnLine
	device.UpdatedAt = time.Now()

	fields := map[string]interface{}{
		"user_id":     device.UserID,
		"status":      device.Status,
		"conn_addr":   device.ConnAddr,
		"client_addr": device.ClientAddr,
		"updated_at":  device.UpdatedAt.Unix(),
	}
	return redisPkg.RedisClient.HSet(ctx, key, fields).Err()
}

// SetDeviceOffline 设置设备离线
func SetDeviceOffline(ctx context.Context, deviceID uint64) error {
	key := deviceInfoKey + strconv.FormatUint(deviceID, 10)
	fields := map[string]interface{}{
		"status":     OffLine,
		"updated_at": time.Now().Unix(),
	}
	return redisPkg.RedisClient.HSet(ctx, key, fields).Err()
}

// GetDeviceOnline 获取设备在线信息
func GetDeviceOnline(ctx context.Context, deviceID uint64) (*repo.Device, error) {
	key := deviceInfoKey + strconv.FormatUint(deviceID, 10)
	ret, err := redisPkg.RedisClient.HGetAll(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	if len(ret) == 0 {
		return nil, nil
	}

	device := &repo.Device{ID: deviceID}
	if userID, err := strconv.ParseUint(ret["user_id"], 10, 64); err == nil {
		device.UserID = userID
	}
	if status, err := strconv.ParseInt(ret["status"], 10, 8); err == nil {
		device.Status = int8(status)
	}
	device.ConnAddr = ret["conn_addr"]
	device.ClientAddr = ret["client_addr"]
	if updatedAt, err := strconv.ParseInt(ret["updated_at"], 10, 64); err == nil {
		device.UpdatedAt = time.Unix(updatedAt, 0)
	}
	return device, nil
}
