package connect

import "sync"

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
