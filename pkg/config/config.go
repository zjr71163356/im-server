// ...existing code...
package config

type Configuration struct {
	Database DatabaseConfig
	Services ServiceConfig
	GRPC     GRPCConfig
}

// DatabaseConfig 封装了所有数据存储的配置
type DatabaseConfig struct {
	MySQL MySQLConfig
	Redis RedisConfig
}

// MySQLConfig 封装了MySQL的配置
type MySQLConfig struct {
	DSN string // DSN (Data Source Name) 是更标准的叫法
}

// RedisConfig 封装了Redis的配置
type RedisConfig struct {
	Host     string
	Password string
}

// ServiceConfig 封装了所有服务监听地址的配置
type ServiceConfig struct {
	Connect ConnectEndpoints
	Logic   LogicEndpoints
	User    UserEndpoints
	File    FileEndpoints
}

// ConnectEndpoints 封装了Connect服务的所有监听端点
type ConnectEndpoints struct {
	LocalAddr string // 对外暴露给其他服务的地址
	RPCAddr   string // RPC监听地址
	TCPAddr   string // TCP长连接监听地址
	WSAddr    string // WebSocket长连接监听地址
}

// LogicEndpoints 封装了Logic服务的监听端点
type LogicEndpoints struct {
	RPCAddr string
}

// UserEndpoints 封装了User服务的监听端点
type UserEndpoints struct {
	RPCAddr string
}

// FileEndpoints 封装了File服务的监听端点
type FileEndpoints struct {
	HTTPAddr string
}

// GRPCConfig 封装所有 gRPC 服务的地址
type GRPCConfig struct {
	ConnectAddr string // Connect 服务的地址
	DeviceAddr  string // Device 服务的地址
	MessageAddr string // Message 服务的地址
	RoomAddr    string // Room 服务的地址
	UserAddr    string // User 服务的地址
}
