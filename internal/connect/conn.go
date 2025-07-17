package connect

import (
	"container/list"
	"im-server/pkg/protocol/pb/connectpb"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

type Transport interface {
	Write([]byte) error
	Close() error
	RemoteAddr() net.Addr
	SetReadDeadline(t time.Time) error
	ReadMessage() ([]byte, error)
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

func (wst *WSTransport) SetReadDeadline(t time.Time) error {
	return wst.Ws.SetReadDeadline(t)
}

func (wst *WSTransport) ReadMessage() ([]byte, error) {
	_, buf, err := wst.Ws.ReadMessage()
	return buf, err

}

func (wst *WSTransport) handleConn() {
	for {
		err := wst.Ws.SetReadDeadline(time.Now().Add(12 * time.Minute))
		if err != nil {
			wst.Ws.Close()
			return
		}
	}
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

func StartWSConn(ws *websocket.Conn, session *Session) {
	conn := &Conn{
		Session:   session,
		Transport: &WSTransport{Ws: ws},
	}
	go conn.Serve()
}

// SignIn 登录
func (c *Conn) SignIn(packet *connectpb.Packet) {}

func (c *Conn) Write(buf []byte) error {
	err := c.Transport.Write(buf)
	if err != nil {
		c.Close()
	}
	return err
}

func (c *Conn) HandleMessage(buf []byte) {
	var packet = new(connectpb.Packet)
	err := proto.Unmarshal(buf, packet)
	if err != nil {
		slog.Error("unmarshal error", "error", err, "len", len(buf))
		return
	}

	if packet.Command != connectpb.Command_SIGN_IN && c.Session.UserID == 0 {
		slog.Error("unauthorized command", "command", packet.Command)
		return
	}
	switch packet.Command {
	case connectpb.Command_SIGN_IN:
		c.SignIn(packet)

	default:
		slog.Error("handler switch other")
	}

}

func (c *Conn) Close() {
	// 取消设备和连接的对应关系
	if c.Session.DeviceID != 0 {
		DeleteConnection(c.Session.DeviceID)
	}

	c.Transport.Close()

}

func (c *Conn) Serve() {
	for {
		err := c.Transport.SetReadDeadline(time.Now().Add(12 * time.Minute))
		if err != nil {
			c.Close()
			return
		}
		data, err := c.Transport.ReadMessage() // 读取消息
		if err != nil {
			c.Close()
			return
		}
		c.HandleMessage(data)
	}
}
