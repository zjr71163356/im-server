#!/bin/bash

# IM服务器停止脚本

echo "停止 IM 服务器..."

# 检查并停止各个服务
if [ -f "auth.pid" ]; then
    AUTH_PID=$(cat auth.pid)
    if kill -0 $AUTH_PID 2>/dev/null; then
        echo "停止认证服务 (PID: $AUTH_PID)..."
        kill $AUTH_PID
    fi
    rm -f auth.pid
fi

if [ -f "user.pid" ]; then
    USER_PID=$(cat user.pid)
    if kill -0 $USER_PID 2>/dev/null; then
        echo "停止用户服务 (PID: $USER_PID)..."
        kill $USER_PID
    fi
    rm -f user.pid
fi

if [ -f "logic.pid" ]; then
    LOGIC_PID=$(cat logic.pid)
    if kill -0 $LOGIC_PID 2>/dev/null; then
        echo "停止逻辑服务 (PID: $LOGIC_PID)..."
        kill $LOGIC_PID
    fi
    rm -f logic.pid
fi

if [ -f "connect.pid" ]; then
    CONNECT_PID=$(cat connect.pid)
    if kill -0 $CONNECT_PID 2>/dev/null; then
        echo "停止连接服务 (PID: $CONNECT_PID)..."
        kill $CONNECT_PID
    fi
    rm -f connect.pid
fi

echo "所有服务已停止!"
