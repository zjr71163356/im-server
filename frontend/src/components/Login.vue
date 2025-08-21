<template>
  <div>
    <h2>Login</h2>
    <form @submit.prevent="handleLogin">
      <label for="username">Username:</label>
      <input id="username" v-model="username" type="text" />

      <label for="password">Password:</label>
      <input id="password" v-model="password" type="password" />

      <button type="submit">Login</button>
    </form>
  </div>
</template>

<script>
import { ref } from "vue";

export default {
  setup() {
    const username = ref("");
    const password = ref("");
    const serverUrl = "ws://127.0.0.1:8080/ws"; // Replace with actual server URL

    const handleLogin = () => {
      const ws = new WebSocket(serverUrl);

      ws.onopen = () => {
        console.log("WebSocket connection established");

        const signInInput = {
          DeviceId: 12345, // Replace with actual device ID
          UserId: username.value,
          Token: password.value,
        };

        const packet = {
          Command: "SIGN_IN",
          RequestId: 1,
          Data: JSON.stringify(signInInput),
        };

        ws.send(JSON.stringify(packet));
      };

      ws.onmessage = (event) => {
        const response = JSON.parse(event.data);
        console.log("Received response:", response);

        if (response.Code === 0) {
          alert("Login successful!");
        } else {
          alert("Login failed: " + response.Message);
        }
      };

      ws.onerror = (error) => {
        console.error("WebSocket error:", error);
        alert("WebSocket connection failed.");
      };

      ws.onclose = () => {
        console.log("WebSocket connection closed");
      };
    };

    return {
      username,
      password,
      handleLogin,
    };
  },
};
</script>

<style scoped>
form {
  display: flex;
  flex-direction: column;
  align-items: center;
}

label {
  margin: 5px 0;
}

input {
  margin: 5px 0;
  padding: 8px;
  width: 200px;
}

button {
  margin-top: 10px;
  padding: 10px 20px;
  background-color: #42b983;
  color: white;
  border: none;
  border-radius: 5px;
  cursor: pointer;
}

button:hover {
  background-color: #369f6e;
}
</style>
