package connect

import (
	"log/slog"
	"sync"

	"im-server/pkg/protocol/pb/connectpb"

	"google.golang.org/protobuf/proto"
)

var ConnectManager = sync.Map{}

func SetConnection(deviceID uint64, conn *Conn) {
	ConnectManager.Store(deviceID, conn)
}

func GetConnection(deviceID uint64) *Conn {
	if conn, ok := ConnectManager.Load(deviceID); ok {
		return conn.(*Conn)
	}
	return nil
}

func DeleteConnection(deviceID uint64) {
	ConnectManager.Delete(deviceID)
}

// DeliverToDevice 将一个 Packet 直接发送到某个设备连接
func DeliverToDevice(deviceID uint64, pkt *connectpb.Packet) bool {
	c := GetConnection(deviceID)
	if c == nil {
		slog.Info("device offline, skip", "deviceID", deviceID)
		return false
	}
	buf, err := proto.Marshal(pkt)
	if err != nil {
		slog.Error("marshal packet", "err", err)
		return false
	}
	if err := c.Write(buf); err != nil {
		slog.Error("write packet", "err", err, "deviceID", deviceID)
		return false
	}
	return true
}

// DeliverToUser 按用户ID向其所有在线设备广播一个 Packet
func DeliverToUser(userID uint64, pkt *connectpb.Packet) int {
	buf, err := proto.Marshal(pkt)
	if err != nil {
		slog.Error("marshal packet", "err", err)
		return 0
	}
	count := 0
	ConnectManager.Range(func(key, value any) bool {
		conn := value.(*Conn)
		if conn.Session != nil && conn.Session.UserID == userID {
			if err := conn.Write(buf); err == nil {
				count++
			} else {
				slog.Error("write packet", "err", err, "deviceID", key)
			}
		}
		return true
	})
	if count == 0 {
		slog.Info("no online devices for user", "userID", userID)
	}
	return count
}
