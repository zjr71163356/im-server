package auth

import (
	"context"
	"encoding/json"
	"fmt"
	redisPkg "im-server/pkg/redis"
	"strconv"
)

const AuthKey = "auth:%d"

type AuthDevice struct {
	DeviceID       uint64 `json:"device_id"`        // 设备ID
	Token          string `json:"token"`            // 设备Token
	TokenExpiresAt int64  `json:"token_expires_at"` // Token过期时间

}

func AuthDeviceGet(userID, deviceID uint64) (*AuthDevice, error) {
	key := fmt.Sprintf(AuthKey, userID)
	bytes, err := redisPkg.RedisClient.HGet(context.Background(), key, strconv.FormatUint(deviceID, 10)).Bytes()
	if err != nil {
		return nil, err
	}

	var device AuthDevice
	err = json.Unmarshal(bytes, &device)
	return &device, err
}

func AuthDeviceSet(userID, deviceID uint64, device AuthDevice) error {
	bytes, err := json.Marshal(device)
	if err != nil {
		return err
	}

	key := fmt.Sprintf(AuthKey, userID)
	_, err = redisPkg.RedisClient.HSet(context.Background(), key, strconv.FormatUint(deviceID, 10), bytes).Result()
	return err
}

func AuthDeviceGetAll(userID uint64) (map[uint64]AuthDevice, error) {
	key := fmt.Sprintf(AuthKey, userID)
	result, err := redisPkg.RedisClient.HGetAll(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}

	var devices = make(map[uint64]AuthDevice, len(result))

	for k, v := range result {
		deviceID, err := strconv.ParseUint(k, 10, 64)
		if err != nil {
			return nil, err
		}

		var device AuthDevice
		err = json.Unmarshal([]byte(v), &device)
		if err != nil {
			return nil, err
		}
		devices[deviceID] = device
	}
	return devices, nil
}
