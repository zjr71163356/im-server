#!/bin/bash

# IM服务器启动脚本

echo "启动 IM 服务器..."

# 检查是否已构建所有服务
if [ ! -f "bin/auth" ] || [ ! -f "bin/connect" ] || [ ! -f "bin/logic" ] || [ ! -f "bin/user" ]; then
    echo "构建服务..."
    make build-all
fi

echo "启动认证服务..."
./bin/auth &
AUTH_PID=$!

echo "启动用户服务..."
./bin/user &
USER_PID=$!

echo "启动逻辑服务..."
./bin/logic &
LOGIC_PID=$!

echo "启动连接服务..."
./bin/connect &
CONNECT_PID=$!

echo "所有服务已启动!"
echo "认证服务 PID: $AUTH_PID (端口: 8020)"
echo "用户服务 PID: $USER_PID (端口: 8030)"
echo "逻辑服务 PID: $LOGIC_PID (端口: 8010)"
echo "连接服务 PID: $CONNECT_PID (端口: 8000, WebSocket: 8002)"

# 创建 PID 文件用于后续停止服务
echo "$AUTH_PID" > auth.pid
echo "$USER_PID" > user.pid
echo "$LOGIC_PID" > logic.pid
echo "$CONNECT_PID" > connect.pid

echo "按 Ctrl+C 停止所有服务..."

# 捕获中断信号并清理
trap 'echo "停止所有服务..."; kill $AUTH_PID $USER_PID $LOGIC_PID $CONNECT_PID; rm -f *.pid; exit' INT

# 等待所有进程
wait
