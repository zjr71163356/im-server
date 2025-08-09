package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"im-server/internal/repo"
	"im-server/pkg/storage"
	"strconv"
)

const AuthKey = "auth:%d"

type authRepo struct{}

var AuthRepo = new(authRepo)

func (*authRepo) Get(userID, deviceID uint64) (*repo.Device, error) {
	key := fmt.Sprintf(AuthKey, userID)
	bytes, err := storage.RedisClient.HGet(context.Background(), key, strconv.FormatUint(deviceID, 10)).Bytes()
	if err != nil {
		return nil, err
	}

	var device repo.Device
	err = json.Unmarshal(bytes, &device)
	return &device, err
}

func (*authRepo) Set(userID, deviceID uint64, device repo.Device) error {
	bytes, err := json.Marshal(device)
	if err != nil {
		return err
	}

	key := fmt.Sprintf(AuthKey, userID)
	_, err = storage.RedisClient.HSet(context.Background(), key, strconv.FormatUint(deviceID, 10), bytes).Result()
	return err
}

func (*authRepo) GetAll(userID uint64) (map[uint64]repo.Device, error) {
	key := fmt.Sprintf(AuthKey, userID)
	result, err := storage.RedisClient.HGetAll(context.Background(), key).Result()
	if err != nil {
		return nil, err
	}

	var devices = make(map[uint64]repo.Device, len(result))

	for k, v := range result {
		deviceID, err := strconv.ParseUint(k, 10, 64)
		if err != nil {
			return nil, err
		}

		var device repo.Device
		err = json.Unmarshal([]byte(v), &device)
		if err != nil {
			return nil, err
		}
		devices[deviceID] = device
	}
	return devices, nil
}
