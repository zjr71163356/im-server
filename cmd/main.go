package cmd

import "im-server/internal/connect"

func main() {

	go connect.StartWSServer(":8080")
}
