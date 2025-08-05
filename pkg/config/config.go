// ...existing code...
package config

var (
	Config Configuration
)

func init() {
	Config = NewConfiguration()
}

func NewConfiguration() Configuration {
	return Configuration{
		Database:   NewDatabaseConfig(),
		Services:   NewServiceConfig(),
		GRPCClient: NewGRPCClientConfig(),
	}
}

func NewDatabaseConfig() DatabaseConfig {
	return DatabaseConfig{
		MySQL: MySQLConfig{"mysql://root:azsx0123456@tcp(localhost:3307)/imserver?multiStatements=true"},
		Redis: RedisConfig{
			Host:     "localhost:6379",
			Password: "",
		},
	}
}

func NewServiceConfig() ServiceConfig {
	return ServiceConfig{
		Connect: ConnectEndpoints{
			LocalAddr: "127.0.0.1:8000",
			RPCAddr:   ":8000",
			TCPAddr:   ":8001",
			WSAddr:    ":8002",
		},
		Logic: LogicEndpoints{RPCAddr: ":8010"},
		User:  UserEndpoints{RPCAddr: ":8020"},
		File:  FileEndpoints{HTTPAddr: ":8030"},
	}
}

func NewGRPCClientConfig() GRPCClientConfig {
	return GRPCClientConfig{
		ConnectTargetAddr: "addrs:///127.0.0.1:8000",
		DeviceTargetAddr:  "addrs:///127.0.0.1:8010",
	}
}

type Configuration struct {
	Database   DatabaseConfig
	Services   ServiceConfig
	GRPCClient GRPCClientConfig
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
type GRPCClientConfig struct {
	ConnectTargetAddr string // Connect 服务的地址
	DeviceTargetAddr  string // Device 服务的地址
	MessageTargetAddr string // Message 服务的地址
	RoomTargetAddr    string // Room 服务的地址
	UserTargetAddr    string // User 服务的地址
}
