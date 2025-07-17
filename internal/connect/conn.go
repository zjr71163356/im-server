package connect

import (
	"container/list"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Transport interface {
	Write([]byte) error
	Close() error
	RemoteAddr() net.Addr
}
type WSTransport struct {
	Mutex sync.Mutex      // WS写锁
	Ws    *websocket.Conn // websocket连接
}

// Write 写入数据
// 以一种线程安全的方式，为一次 WebSocket
// 消息写入操作设置一个短暂的超时（10毫秒）
// ，然后将给定的二进制数据 (buf) 发送给客户端。
func (wst *WSTransport) Write(buf []byte) error {
	wst.Mutex.Lock()
	defer wst.Mutex.Unlock()
	err := wst.Ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return err
	}
	return wst.Ws.WriteMessage(websocket.BinaryMessage, buf)
}

// 使用WebSocket的方法进行连接关闭
func (wst *WSTransport) Close() error {
	return wst.Ws.Close()

}

func (wst *WSTransport) RemoteAddr() net.Addr {
	return wst.Ws.RemoteAddr()
}

type Session struct {
	UserID   uint64        // 用户ID
	DeviceID uint64        // 设备ID
	RoomID   uint64        // 订阅的房间ID
	Element  *list.Element // 链表节点
}

type Conn struct {
	Session   *Session
	Transport Transport
}

func NewWSConnection(ws *websocket.Conn, session *Session) *Conn {
	return &Conn{
		Session:   session,
		Transport: &WSTransport{Ws: ws},
	}
}

func (c *Conn) Write(buf []byte) error {
	var err error
	err = c.Transport.Write(buf)
	if err != nil {
		c.Close()
	}
	return err
}

func (c *Conn) Close() {
	// 取消设备和连接的对应关系
	if c.Session.DeviceID != 0 {
		DeleteConnection(c.Session.DeviceID)
	}

	c.Transport.Close()

}
