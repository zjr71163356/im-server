package connect

import (
	"container/list"
	"sync"

	"github.com/gorilla/websocket"
)

type Connection interface {
	Close(err error)
	GetAddr() string
	HandleMessage(buf []byte)
	Write(buf []byte) error
}
type ConnContext struct {
	UserID   uint64        // 用户ID
	DeviceID uint64        // 设备ID
	RoomID   uint64        // 订阅的房间ID
	Element  *list.Element // 链表节点
}

type WebSocketConnection struct {
	WSMutex sync.Mutex      // WS写锁
	WS      *websocket.Conn // websocket
	ConnContext
}

func (cc *ConnContext) Close(err error) {
	// Implement the logic to close the connection with the provided error
}


