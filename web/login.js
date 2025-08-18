class IMClient {
  constructor() {
    this.websocket = null;
    this.isConnected = false;
    this.connectionStatus = document.getElementById("connectionStatus");
    this.statusMessage = document.getElementById("statusMessage");
    this.loginBtn = document.getElementById("loginBtn");
    this.loginForm = document.getElementById("loginForm");

    this.initEventListeners();
  }

  initEventListeners() {
    this.loginForm.addEventListener("submit", (e) => {
      e.preventDefault();
      this.handleLogin();
    });
  }

  updateConnectionStatus(connected, message = "") {
    this.isConnected = connected;
    this.connectionStatus.textContent = connected ? "已连接" : "未连接";
    this.connectionStatus.className = `connection-status ${
      connected ? "connected" : "disconnected"
    }`;

    if (message) {
      this.showStatusMessage(message, connected ? "success" : "error");
    }
  }

  showStatusMessage(message, type = "info") {
    this.statusMessage.innerHTML = `<div class="status-${type}">${message}</div>`;

    // 自动清除成功消息
    if (type === "success") {
      setTimeout(() => {
        this.statusMessage.innerHTML = "";
      }, 3000);
    }
  }

  async connectWebSocket(serverUrl) {
    return new Promise((resolve, reject) => {
      try {
        this.websocket = new WebSocket(serverUrl);

        this.websocket.onopen = () => {
          console.log("WebSocket连接已建立");
          this.updateConnectionStatus(true, "WebSocket连接成功");
          resolve();
        };

        this.websocket.onmessage = (event) => {
          this.handleMessage(event);
        };

        this.websocket.onclose = (event) => {
          console.log("WebSocket连接已关闭", event);
          this.updateConnectionStatus(false, "WebSocket连接已关闭");
          this.websocket = null;
        };

        this.websocket.onerror = (error) => {
          console.error("WebSocket错误:", error);
          this.updateConnectionStatus(false, "WebSocket连接失败");
          reject(error);
        };

        // 设置连接超时
        setTimeout(() => {
          if (this.websocket && this.websocket.readyState !== WebSocket.OPEN) {
            this.websocket.close();
            reject(new Error("连接超时"));
          }
        }, 5000);
      } catch (error) {
        reject(error);
      }
    });
  }

  handleMessage(event) {
    try {
      console.log("收到消息:", event.data);

      // 检查是否是二进制数据
      if (event.data instanceof ArrayBuffer) {
        console.log("收到二进制数据，长度:", event.data.byteLength);
        console.log(
          "数据内容 (hex):",
          Array.from(new Uint8Array(event.data))
            .map((b) => b.toString(16).padStart(2, "0"))
            .join(" ")
        );

        const response = this.parseProtobufResponse(event.data);
        console.log("解析后的响应:", response);
        this.handleSignInResponse(response);
      } else if (event.data instanceof Blob) {
        // 处理 Blob 数据
        event.data.arrayBuffer().then((buffer) => {
          console.log(
            "收到Blob数据，转换为ArrayBuffer，长度:",
            buffer.byteLength
          );
          console.log(
            "数据内容 (hex):",
            Array.from(new Uint8Array(buffer))
              .map((b) => b.toString(16).padStart(2, "0"))
              .join(" ")
          );

          const response = this.parseProtobufResponse(buffer);
          console.log("解析后的响应:", response);
          this.handleSignInResponse(response);
        });
      } else {
        console.log("收到文本消息:", event.data);
      }
    } catch (error) {
      console.error("处理消息错误:", error);
      this.showStatusMessage("处理服务器响应时出错", "error");
    }
  }

  // Protobuf 解析 - 实现基本的 wire format 解析
  parseProtobufResponse(buffer) {
    try {
      const data = new Uint8Array(buffer);
      let offset = 0;
      const packet = {
        command: 0,
        requestId: 0,
        code: 0,
        message: "",
        data: null,
      };

      while (offset < data.length) {
        // 读取字段标签
        const tag = this.decodeVarint(data, offset);
        offset = tag.offset;

        const fieldNumber = tag.value >>> 3;
        const wireType = tag.value & 0x7;

        switch (fieldNumber) {
          case 1: // command
            if (wireType === 0) {
              const result = this.decodeVarint(data, offset);
              packet.command = result.value;
              offset = result.offset;
            }
            break;
          case 2: // request_id
            if (wireType === 0) {
              const result = this.decodeVarint(data, offset);
              packet.requestId = result.value;
              offset = result.offset;
            }
            break;
          case 3: // code
            if (wireType === 0) {
              const result = this.decodeVarint(data, offset);
              packet.code = result.value;
              offset = result.offset;
            }
            break;
          case 4: // message
            if (wireType === 2) {
              const result = this.decodeString(data, offset);
              packet.message = result.value;
              offset = result.offset;
            }
            break;
          case 5: // data
            if (wireType === 2) {
              const result = this.decodeBytes(data, offset);
              packet.data = result.value;
              offset = result.offset;
            }
            break;
          default:
            // 跳过未知字段
            offset = this.skipField(data, offset, wireType);
            break;
        }
      }

      return packet;
    } catch (error) {
      console.error("解析Protobuf错误:", error);
      return {
        command: 0,
        requestId: 0,
        code: -1,
        message: "解析错误",
        data: null,
      };
    }
  }

  // 解码 varint
  decodeVarint(data, offset) {
    let value = 0;
    let shift = 0;
    let currentOffset = offset;

    while (currentOffset < data.length) {
      const byte = data[currentOffset++];
      value |= (byte & 0x7f) << shift;
      if ((byte & 0x80) === 0) {
        break;
      }
      shift += 7;
      if (shift >= 64) {
        throw new Error("Varint too long");
      }
    }

    return { value, offset: currentOffset };
  }

  // 解码字符串
  decodeString(data, offset) {
    const lengthResult = this.decodeVarint(data, offset);
    const length = lengthResult.value;
    const startOffset = lengthResult.offset;
    const endOffset = startOffset + length;

    if (endOffset > data.length) {
      throw new Error("String length exceeds buffer");
    }

    const bytes = data.slice(startOffset, endOffset);
    const value = new TextDecoder().decode(bytes);

    return { value, offset: endOffset };
  }

  // 解码字节数组
  decodeBytes(data, offset) {
    const lengthResult = this.decodeVarint(data, offset);
    const length = lengthResult.value;
    const startOffset = lengthResult.offset;
    const endOffset = startOffset + length;

    if (endOffset > data.length) {
      throw new Error("Bytes length exceeds buffer");
    }

    const value = data.slice(startOffset, endOffset);

    return { value, offset: endOffset };
  }

  // 跳过字段
  skipField(data, offset, wireType) {
    switch (wireType) {
      case 0: // varint
        return this.decodeVarint(data, offset).offset;
      case 1: // fixed64
        return offset + 8;
      case 2: // length-delimited
        const lengthResult = this.decodeVarint(data, offset);
        return lengthResult.offset + lengthResult.value;
      case 5: // fixed32
        return offset + 4;
      default:
        throw new Error(`Unknown wire type: ${wireType}`);
    }
  }

  handleSignInResponse(response) {
    console.log("收到SignIn响应:", response);

    if (response.code === 0) {
      this.showStatusMessage("登录成功！", "success");
      this.loginBtn.textContent = "已登录";
      this.loginBtn.disabled = true;
    } else {
      this.showStatusMessage(`登录失败: 错误码 ${response.code}`, "error");
      this.loginBtn.disabled = false;
      this.loginBtn.textContent = "重试登录";
    }
  }

  // Protobuf 编码 - 实现基本的 wire format
  createSignInPacket(userId, deviceId, token) {
    // 编码 SignInInput
    const signInInputData = this.encodeSignInInput(userId, deviceId, token);

    // 编码 Packet
    return this.encodePacket(1, Date.now(), 0, "", signInInputData);
  }

  // 编码 varint (protobuf 变长整数)
  encodeVarint(value) {
    const result = [];
    while (value >= 0x80) {
      result.push((value & 0xff) | 0x80);
      value >>>= 7;
    }
    result.push(value & 0xff);
    return new Uint8Array(result);
  }

  // 编码 uint64
  encodeUint64(value) {
    const result = [];
    let num = BigInt(value);
    while (num >= 0x80n) {
      result.push(Number((num & 0xffn) | 0x80n));
      num >>= 7n;
    }
    result.push(Number(num & 0xffn));
    return new Uint8Array(result);
  }

  // 编码字符串
  encodeString(str) {
    const bytes = new TextEncoder().encode(str);
    const length = this.encodeVarint(bytes.length);
    const result = new Uint8Array(length.length + bytes.length);
    result.set(length, 0);
    result.set(bytes, length.length);
    return result;
  }

  // 编码字节数组
  encodeBytes(bytes) {
    const length = this.encodeVarint(bytes.length);
    const result = new Uint8Array(length.length + bytes.length);
    result.set(length, 0);
    result.set(bytes, length.length);
    return result;
  }

  // 编码 SignInInput 消息
  encodeSignInInput(userId, deviceId, token) {
    const parts = [];

    // field 1: device_id (uint64)
    parts.push(new Uint8Array([0x08])); // field number 1, wire type 0 (varint)
    parts.push(this.encodeUint64(deviceId));

    // field 2: user_id (uint64)
    parts.push(new Uint8Array([0x10])); // field number 2, wire type 0 (varint)
    parts.push(this.encodeUint64(userId));

    // field 3: token (string)
    parts.push(new Uint8Array([0x1a])); // field number 3, wire type 2 (length-delimited)
    parts.push(this.encodeString(token));

    // 计算总长度并合并
    const totalLength = parts.reduce((sum, part) => sum + part.length, 0);
    const result = new Uint8Array(totalLength);
    let offset = 0;
    for (const part of parts) {
      result.set(part, offset);
      offset += part.length;
    }

    return result;
  }

  // 编码 Packet 消息
  encodePacket(command, requestId, code, message, data) {
    const parts = [];

    // field 1: command (enum/int32)
    if (command !== 0) {
      parts.push(new Uint8Array([0x08])); // field number 1, wire type 0
      parts.push(this.encodeVarint(command));
    }

    // field 2: request_id (int64)
    if (requestId !== 0) {
      parts.push(new Uint8Array([0x10])); // field number 2, wire type 0
      parts.push(this.encodeUint64(requestId));
    }

    // field 3: code (int32)
    if (code !== 0) {
      parts.push(new Uint8Array([0x18])); // field number 3, wire type 0
      parts.push(this.encodeVarint(code));
    }

    // field 4: message (string)
    if (message && message.length > 0) {
      parts.push(new Uint8Array([0x22])); // field number 4, wire type 2
      parts.push(this.encodeString(message));
    }

    // field 5: data (bytes)
    if (data && data.length > 0) {
      parts.push(new Uint8Array([0x2a])); // field number 5, wire type 2
      parts.push(this.encodeBytes(data));
    }

    // 计算总长度并合并
    const totalLength = parts.reduce((sum, part) => sum + part.length, 0);
    const result = new Uint8Array(totalLength);
    let offset = 0;
    for (const part of parts) {
      result.set(part, offset);
      offset += part.length;
    }

    return result.buffer;
  }

  async sendSignInPacket(userId, deviceId, token) {
    if (!this.websocket || this.websocket.readyState !== WebSocket.OPEN) {
      throw new Error("WebSocket未连接");
    }

    const packet = this.createSignInPacket(userId, deviceId, token);

    console.log("发送SignIn包:", {
      userId: userId,
      deviceId: deviceId,
      token: token,
      packetSize: packet.byteLength,
      packetHex: Array.from(new Uint8Array(packet))
        .map((b) => b.toString(16).padStart(2, "0"))
        .join(" "),
    });

    this.websocket.send(packet);
  }

  async handleLogin() {
    const serverUrl = document.getElementById("serverUrl").value.trim();
    const userId = parseInt(document.getElementById("userId").value);
    const deviceId = parseInt(document.getElementById("deviceId").value);
    const token = document.getElementById("token").value.trim();

    // 验证输入
    if (!serverUrl || !userId || !deviceId || !token) {
      this.showStatusMessage("请填写完整的登录信息", "error");
      return;
    }

    this.loginBtn.disabled = true;
    this.loginBtn.textContent = "连接中...";

    try {
      // 先关闭现有连接
      if (this.websocket) {
        this.websocket.close();
        this.websocket = null;
      }

      this.showStatusMessage("正在连接WebSocket...", "info");

      // 建立WebSocket连接
      await this.connectWebSocket(serverUrl);

      this.loginBtn.textContent = "登录中...";
      this.showStatusMessage("正在发送登录请求...", "info");

      // 发送登录请求
      await this.sendSignInPacket(userId, deviceId, token);

      // 等待响应的逻辑在handleMessage中处理
    } catch (error) {
      console.error("登录失败:", error);
      this.showStatusMessage(`登录失败: ${error.message}`, "error");
      this.loginBtn.disabled = false;
      this.loginBtn.textContent = "重试登录";
    }
  }

  // 断开连接
  disconnect() {
    if (this.websocket) {
      this.websocket.close();
      this.websocket = null;
    }
    this.updateConnectionStatus(false, "已断开连接");
    this.loginBtn.disabled = false;
    this.loginBtn.textContent = "连接并登录";
  }
}

// 初始化客户端
const imClient = new IMClient();

// 添加页面卸载时的清理
window.addEventListener("beforeunload", () => {
  imClient.disconnect();
});

// 添加一些调试功能
window.imClient = imClient; // 方便在控制台调试
