package auth

import (
	"context"
	"fmt"
	"time"
)

func Auth(ctx context.Context, userID, deviceID uint64, token string) error {
	authDevice, err := AuthDeviceGet(userID, deviceID)
	if err != nil {
		return err
	}

	if authDevice.TokenExpiresAt < time.Now().Unix() {
		// Token 已过期，可能需要重新认证
		return fmt.Errorf("token expired for user %d on device %d", userID, deviceID)
	}

	if authDevice.Token != token {
		// Token 不匹配，认证失败
		return fmt.Errorf("invalid token for user %d on device %d", userID, deviceID)
	}
	// 认证成功，更新设备状态为在线

	return nil
}
