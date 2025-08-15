package main

import (
	"im-server/internal/connect"
	"im-server/pkg/config"
	"im-server/pkg/protocol/pb/connectpb"
	"sync"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

func main() {
	// 初始化配置
	config.Config.Services.Connect.LocalAddr = "127.0.0.1:8080"

	// 模拟一个 WebSocket 连接
	ws := &connect.WSTransport{
		Mutex: sync.Mutex{},
		Ws:    &websocket.Conn{
			
		}, // 这里需要一个实际的 websocket.Conn
	}

	// 创建一个 Session
	session := &connect.Session{}

	// 创建一个 Conn
	conn := &connect.Conn{
		Session:   session,
		Transport: ws,
	}

	// 模拟一个 SignInInput
	signInInput := &connectpb.SignInInput{
		DeviceId: 12345,
		UserId:   67890,
		Token:    "test-token",
	}
	data, _ := proto.Marshal(signInInput)

	// 创建一个 Packet
	packet := &connectpb.Packet{
		Command: connectpb.Command_SIGN_IN,
		Data:    data,
	}

	// 调用 SignIn
	conn.SignIn(packet)
}
