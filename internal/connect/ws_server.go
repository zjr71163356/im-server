package connect

import (
	"log/slog"
	"net/http"

	"im-server/pkg/config"
	"im-server/pkg/jwt"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 65536,
	CheckOrigin: func(r *http.Request) bool {
		// 允许跨域，生产环境按需收紧
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("wsHandler has been called, attempting to upgrade connection...")
	// 1) 从查询参数读取 token 并校验
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}
	uid, did, err := jwt.ParseJWT(token, []byte(config.Config.JWT.Secret), config.Config.JWT.Issuer, config.Config.JWT.Audience)
	if err != nil {
		slog.Error("invalid jwt", "err", err)
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// 2) 升级为 WebSocket
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("upgrade failed", "err", err)
		return
	}

	// 3) 构建已认证会话并启动连接（StartWSConn 会在 session 带 deviceID 时完成注册）
	StartWSConn(wsConn, &Session{UserID: uid, DeviceID: did})

}

func StartWSServer(addr string) {
	http.HandleFunc("/ws", wsHandler)
	slog.Info("websocket server running", "addr", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		slog.Error("start ws server", "err", err)
	}
}
