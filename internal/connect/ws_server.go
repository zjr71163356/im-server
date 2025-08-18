package connect

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 65536,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all connections for simplicity, adjust as needed
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	slog.Info("wsHandler has been called, attempting to upgrade connection...")
	// Handle WebSocket connections
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("Failed to upgrade connection", "error", err)
		return
	}
	StartWSConn(wsConn, &Session{})

}

func StartWSServer(addr string) {
	http.HandleFunc("/ws", wsHandler)
	slog.Info("websocket server running")
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		slog.Error("Failed to start WebSocket server", "error", err)
		return
	}

}
