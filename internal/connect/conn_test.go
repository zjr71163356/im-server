package connect

import (
	"im-server/pkg/config"
	"im-server/pkg/mocks"
	"im-server/pkg/protocol/pb/connectpb"
	"im-server/pkg/rpc"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

// TestWebSocketToSignInFlow 测试完整的 WebSocket 连接到 SignIn 的流程
func TestWebSocketToSignInFlow(t *testing.T) {
	// 1. 初始化配置和 mock
	config.Config.Services.Connect.LocalAddr = "127.0.0.1:8080"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDeviceIntServiceClient(ctrl)
	rpc.SetDeviceIntServiceClient(mockClient)

	// 模拟 gRPC 调用成功
	mockClient.EXPECT().ConnSignIn(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	mockClient.EXPECT().Offline(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	// 2. 启动真实的 StartWSServer 在后台，使用一个临时可用端口
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	go StartWSServer(addr)
	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)
	// 3. 发起Http请求连接到 WebSocket
	// 构造 WebSocket URL，客户端通过这个地址发起 HTTP GET 请求进行协议升级
	wsURL := "ws://" + addr + "/ws"

	// websocket.DefaultDialer.Dial 会发起一个 HTTP 请求来建立连接
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("发起 HTTP Upgrade 请求建立 WebSocket 连接失败: %v", err)
	}
	defer ws.Close()

	// 4. 准备 SignIn 消息
	signInInput := &connectpb.SignInInput{
		DeviceId: 12345,
		UserId:   67890,
		Token:    "test-token",
	}

	signInData, err := proto.Marshal(signInInput)
	if err != nil {
		t.Fatalf("序列化 SignInInput 失败: %v", err)
	}

	packet := &connectpb.Packet{
		Command:   connectpb.Command_SIGN_IN,
		RequestId: 1,
		Data:      signInData,
	}

	packetData, err := proto.Marshal(packet)
	if err != nil {
		t.Fatalf("序列化 Packet 失败: %v", err)
	}

	// 5. 通过已建立的 WebSocket 连接发送消息
	err = ws.WriteMessage(websocket.BinaryMessage, packetData)
	if err != nil {
		t.Fatalf("通过 WebSocket 发送消息失败: %v", err)
	}

	// 6. 接收响应消息
	ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, responseData, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("读取 WebSocket 响应失败: %v", err)
	}

	// 7. 验证响应
	responsePacket := &connectpb.Packet{}
	err = proto.Unmarshal(responseData, responsePacket)
	if err != nil {
		t.Fatalf("反序列化响应 Packet 失败: %v", err)
	}

	// 验证响应包的基本信息
	if responsePacket.RequestId != packet.RequestId {
		t.Errorf("期望 RequestId %d, 得到 %d", packet.RequestId, responsePacket.RequestId)
	}

	if responsePacket.Code != 0 {
		t.Errorf("期望成功码 0, 得到 %d", responsePacket.Code)
	}
	t.Logf("收到响应 Packet: %+v", responsePacket)
	t.Logf("WebSocket to SignIn 流程测试通过")
}
