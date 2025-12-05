package device

import (
	"context"
	"im-server/pkg/dao"
	redisPkg "im-server/pkg/redis"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	deviceInfoKey = "device:info:"
	OnLine        = 1 // 设备在线
	OffLine       = 0 // 设备离线

	// 在线记录的有效期，心跳或读写消息时会续期，防止“假在线”
	DeviceOnlineTTL = 5 * time.Minute
	// 离线记录的存活时间，保留一段时间便于排查/查询
	deviceOfflineTTL = 24 * time.Hour
)

func deviceKey(deviceID uint64) string {
	return deviceInfoKey + strconv.FormatUint(deviceID, 10)
}

// SetDeviceOnline 设置设备在线信息到redis
func SetDeviceOnline(ctx context.Context, device *dao.Device) error {
	key := deviceKey(device.ID)
	device.Status = OnLine
	device.UpdatedAt = time.Now()
	expiresAt := device.UpdatedAt.Add(DeviceOnlineTTL)

	fields := map[string]interface{}{
		"user_id":     device.UserID,
		"status":      device.Status,
		"conn_addr":   device.ConnAddr,
		"client_addr": device.ClientAddr,
		"updated_at":  device.UpdatedAt.Unix(),
		"expires_at":  expiresAt.Unix(),
	}
	pipe := redisPkg.RedisClient.TxPipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, DeviceOnlineTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// TouchDeviceOnline 续期在线态，避免连接异常中断导致“假在线”
func TouchDeviceOnline(ctx context.Context, deviceID, userID uint64, connAddr, clientAddr string) error {
	key := deviceKey(deviceID)
	now := time.Now()
	fields := map[string]interface{}{
		"user_id":     userID,
		"status":      OnLine,
		"conn_addr":   connAddr,
		"client_addr": clientAddr,
		"updated_at":  now.Unix(),
		"expires_at":  now.Add(DeviceOnlineTTL).Unix(),
	}
	pipe := redisPkg.RedisClient.TxPipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, DeviceOnlineTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// SetDeviceOffline 设置设备离线（幂等：多次调用只会覆盖为离线并刷新过期时间）
func SetDeviceOffline(ctx context.Context, deviceID uint64) error {
	key := deviceKey(deviceID)
	now := time.Now()
	fields := map[string]interface{}{
		"status":     OffLine,
		"updated_at": now.Unix(),
		"expires_at": now.Add(deviceOfflineTTL).Unix(),
	}
	pipe := redisPkg.RedisClient.TxPipeline()
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, deviceOfflineTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// GetDeviceOnline 获取设备在线信息
func GetDeviceOnline(ctx context.Context, deviceID uint64) (*dao.Device, error) {
	key := deviceKey(deviceID)
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

	device := &dao.Device{ID: deviceID}
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
