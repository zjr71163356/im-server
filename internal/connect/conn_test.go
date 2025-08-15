package connect

import (
	"net"
	"testing"
	"time"

	"im-server/pkg/config"
	"im-server/pkg/mocks"
	"im-server/pkg/protocol/pb/connectpb"
	"im-server/pkg/rpc"

	"github.com/golang/mock/gomock"
	"google.golang.org/protobuf/proto"
)

// MockTransport 是一个模拟的 Transport 实现，用于测试
type MockTransport struct{}

func (m *MockTransport) Write([]byte) error                { return nil }
func (m *MockTransport) Close() error                      { return nil }
func (m *MockTransport) RemoteAddr() net.Addr              { return &net.IPAddr{IP: net.ParseIP("127.0.0.1")} }
func (m *MockTransport) SetReadDeadline(t time.Time) error { return nil }
func (m *MockTransport) ReadMessage() ([]byte, error)      { return nil, nil }

func TestSignIn(t *testing.T) {
	// 初始化配置
	config.Config.Services.Connect.LocalAddr = "127.0.0.1:8080"

	// 创建一个模拟的 gRPC 客户端
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockDeviceIntServiceClient(ctrl)
	rpc.SetDeviceIntServiceClient(mockClient)
	// rpc.SetDeviceIntServiceClient(mockClient)

	// 模拟 gRPC 的 ConnSignIn 调用
	mockClient.EXPECT().ConnSignIn(gomock.Any(), gomock.Any()).Return(nil, nil)

	// 创建一个测试用的 Packet
	signInInput := &connectpb.SignInInput{
		DeviceId: 12345,
		UserId:   67890,
		Token:    "test-token",
	}
	data, err := proto.Marshal(signInInput)
	if err != nil {
		t.Fatalf("failed to marshal SignInInput: %v", err)
	}

	packet := &connectpb.Packet{
		Command: connectpb.Command_SIGN_IN,
		Data:    data,
	}

	// 创建一个 Conn 对象
	session := &Session{}
	conn := &Conn{
		Session:   session,
		Transport: &MockTransport{},
	}

	// 调用 SignIn 函数
	conn.SignIn(packet)

	// 验证结果
	if conn.Session.DeviceID != signInInput.DeviceId {
		t.Errorf("expected DeviceID %d, got %d", signInInput.DeviceId, conn.Session.DeviceID)
	}
	if conn.Session.UserID != signInInput.UserId {
		t.Errorf("expected UserID %d, got %d", signInInput.UserId, conn.Session.UserID)
	}
}
