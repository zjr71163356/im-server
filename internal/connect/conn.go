package connect

import (
	"container/list"
	"im-server/pkg/protocol/pb/connectpb"
	"im-server/pkg/rpc"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// Transport 是一个接口，抽象了所有底层网络传输（如TCP、WebSocket）必须具备的通用能力。
type Transport interface {
	Write([]byte) error
	Close() error
	RemoteAddr() net.Addr
	SetReadDeadline(t time.Time) error
	ReadMessage() ([]byte, error)
}

// WSTransport 是 Transport 接口针对 WebSocket 的具体实现。
type WSTransport struct {
	Mutex sync.Mutex      // WS写锁，保证并发写入的线程安全
	Ws    *websocket.Conn // 底层的 websocket 连接
}

// Write 以一种线程安全的方式，为一次 WebSocket 消息写入操作设置一个短暂的超时，
// 然后将给定的二进制数据 (buf) 发送给客户端。
func (wst *WSTransport) Write(buf []byte) error {
	wst.Mutex.Lock()
	defer wst.Mutex.Unlock()
	err := wst.Ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return err
	}
	return wst.Ws.WriteMessage(websocket.BinaryMessage, buf)
}

// Close 关闭底层的 WebSocket 连接。
func (wst *WSTransport) Close() error {
	return wst.Ws.Close()
}

// RemoteAddr 返回 WebSocket 连接的远端网络地址。
func (wst *WSTransport) RemoteAddr() net.Addr {
	return wst.Ws.RemoteAddr()
}

// SetReadDeadline 设置底层连接的读取超时时间。
func (wst *WSTransport) SetReadDeadline(t time.Time) error {
	return wst.Ws.SetReadDeadline(t)
}

// ReadMessage 从 WebSocket 连接中读取一条完整的消息。
func (wst *WSTransport) ReadMessage() ([]byte, error) {
	_, buf, err := wst.Ws.ReadMessage()
	return buf, err
}

// Session 存储了一个逻辑连接的会话信息，如用户和设备标识。
type Session struct {
	UserID   uint64        // 用户ID
	DeviceID uint64        // 设备ID
	RoomID   uint64        // 订阅的房间ID
	Element  *list.Element // 在管理器链表中的节点，方便快速删除
}

// Conn 代表一个抽象的逻辑连接，它包含会话信息和一个具体的 Transport 实现。
type Conn struct {
	Session   *Session
	Transport Transport
}

// StartWSConn 是处理新 WebSocket 连接的入口函数。
// 它创建一个 Conn 对象，并启动一个 goroutine 来服务于这个连接。
func StartWSConn(ws *websocket.Conn, session *Session) {
	conn := &Conn{
		Session:   session,
		Transport: &WSTransport{Ws: ws},
	}
	go conn.Serve()
}

// 这个 func (c *Conn) SignIn(packet *pb.Packet) 是整个用户连接生命周期中最关键的第一个业务步骤。
// 它的核心职责是处理客户端的登录请求，验证其身份，
// 并在验证成功后，将这个匿名的网络连接与一个具体的用户身份绑定起来。
func (c *Conn) SignIn(packet *connectpb.Packet) {
	// TODO: 实现登录逻辑，例如验证 token，解析 SignInInput，并更新 Session
	var signInputReq connectpb.SignInInput
	err := proto.Unmarshal(packet.Data, &signInputReq)
	if err != nil {
		slog.Error("unmarshal error", "error", err)
		return
	}

	//TODO
	//使用gRPC进行远程调用函数验证登录
	//需要验证传入的信息是否与数据库中的符合，必然涉及repo的开发，目前先不加(7.21)
	rpc.GetDeviceIntServiceClient().ConnSignIn()
	c.Send(packet, nil, err)

	c.Session.DeviceID = signInputReq.DeviceId
	c.Session.UserID = signInputReq.UserId

	SetConnection(c.Session.DeviceID, c)

	// 验证 token，更新 Session 等逻辑
}

func (c *Conn) Send(packet *connectpb.Packet, message proto.Message, err error) {

	packet.Data = nil // 这里可以根据需要设置数据
	packet.Code = 0
	packet.Message = ""

	if err != nil {

	}

	if message != nil {
		data, err := proto.Marshal(message)
		if err != nil {
			slog.Error("marshal error", "error", err)
			return
		}
		packet.Data = data
	}

	buf, err := proto.Marshal(packet)
	if err != nil {
		slog.Error("marshal error", "error", err)
		return
	}
	err = c.Write(buf)
	if err != nil {
		slog.Error("write error", "error", err)
		return
	}

	slog.Info("send packet", "packet", packet, "message", message, "error", err)

}

// Write 向连接写入数据。如果写入失败，则关闭连接。
func (c *Conn) Write(buf []byte) error {
	err := c.Transport.Write(buf)
	if err != nil {
		c.Close()
	}
	return err
}

// HandleMessage 是中心消息处理器。它反序列化收到的数据包，并根据指令分发给不同的处理函数。
func (c *Conn) HandleMessage(buf []byte) {
	var packet = new(connectpb.Packet)
	err := proto.Unmarshal(buf, packet)
	if err != nil {
		slog.Error("unmarshal error", "error", err, "len", len(buf))
		return
	}

	slog.Debug("HandleMessage", "packet", packet)

	// 检查除了登录指令外的所有请求是否已经认证
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

// Close 关闭一个连接，并执行相关的清理工作。
func (c *Conn) Close() {

	// 如果设备已登录，则从全局连接管理器中删除此连接
	if c.Session.DeviceID != 0 {
		DeleteConnection(c.Session.DeviceID)
	}

	//TO DO
	// 取消订阅房间

	//TO DO
	// gPRC远程调用函数，使得设备离线

	// 关闭底层的物理连接
	c.Transport.Close()
}

// Serve 是每个连接的主服务循环。
// 它在一个无限循环中不断地设置超时、读取消息，并将消息分发给 HandleMessage 处理。
// 当发生任何错误时，循环终止，连接被关闭。
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
