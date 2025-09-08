package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var (
	// Config 是一个全局变量，持有所有应用程序的配置
	Config Configuration
)

// init 函数在程序启动时自动运行，加载配置文件
func init() {
	// 注意：为了简单起见，这里硬编码了配置文件路径。
	// 在生产环境中，最好通过命令行标志或环境变量来传递路径。
	err := loadConfigFromFile("/home/tyrfly/im-server/config.yaml")
	if err != nil {
		// 配置是程序运行的基础，如果加载失败，则直接 panic
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}
}

// loadConfigFromFile 是一个内部函数，用于加载和解析配置
func loadConfigFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}

	var tempConfig Configuration
	err = yaml.Unmarshal(data, &tempConfig)
	if err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 动态生成 gRPC 客户端配置
	tempConfig.GRPCClient = generateGRPCClientConfig(tempConfig.Services)
	Config = tempConfig
	return nil
}

// LoadConfig 加载配置文件 (为了保持接口兼容性，但现在主要逻辑在 init 中)
func LoadConfig(path string) (*Configuration, error) {
	err := loadConfigFromFile(path)
	if err != nil {
		return nil, err
	}
	return &Config, nil
}

// generateGRPCClientConfig 根据加载的服务配置动态生成 gRPC 客户端的目标地址
func generateGRPCClientConfig(services ServiceConfig) GRPCClientConfig {
	return GRPCClientConfig{
		ConnectTargetAddr: fmt.Sprintf("addrs:///%s", services.Connect.LocalAddr),
		DeviceTargetAddr:  fmt.Sprintf("addrs:///%s", services.Device.LocalAddr), // Device 服务地址
		AuthTargetAddr:    fmt.Sprintf("addrs:///%s", services.Auth.LocalAddr),
	}
}

// Configuration 是所有配置的根结构体
type Configuration struct {
	Database   DatabaseConfig   `yaml:"database"`
	Services   ServiceConfig    `yaml:"services"`
	GRPCClient GRPCClientConfig `yaml:"-"` // 通过代码动态生成，忽略 YAML 解析
}

// DatabaseConfig 封装了所有数据存储的配置
type DatabaseConfig struct {
	MySQL MySQLConfig `yaml:"mysql"`
	Redis RedisConfig `yaml:"redis"`
}

// MySQLConfig 封装了MySQL的配置
type MySQLConfig struct {
	DSN string `yaml:"dsn"`
}

// RedisConfig 封装了Redis的配置
type RedisConfig struct {
	Host     string `yaml:"host"`
	Password string `yaml:"password"`
}

// ServiceConfig 封装了所有服务监听地址的配置
type ServiceConfig struct {
	Connect ConnectEndpoints `yaml:"connect"`
	Device  DeviceEndpoints  `yaml:"device"`
	Auth    AuthEndpoints    `yaml:"auth"`
	User    UserEndpoints    `yaml:"user"`
	File    FileEndpoints    `yaml:"file"`
	Gateway GatewayEndpoints `yaml:"gateway"`
}

// ConnectEndpoints 封装了Connect服务的所有监听端点
type ConnectEndpoints struct {
	LocalAddr string `yaml:"local_addr"` // 对外暴露给其他服务的地址
	RPCAddr   string `yaml:"rpc_addr"`   // RPC监听地址
	TCPAddr   string `yaml:"tcp_addr"`   // TCP长连接监听地址
	WSAddr    string `yaml:"ws_addr"`    // WebSocket长连接监听地址
}

// DeviceEndpoints 封装了Device服务的监听端点
type DeviceEndpoints struct {
	LocalAddr string `yaml:"local_addr"`
	RPCAddr   string `yaml:"rpc_addr"`
}

// UserEndpoints 封装了User服务的监听端点
type UserEndpoints struct {
	LocalAddr string `yaml:"local_addr"`
	RPCAddr   string `yaml:"rpc_addr"`
}

// FileEndpoints 封装了File服务的监听端点
type FileEndpoints struct {
	HTTPAddr string `yaml:"http_addr"`
}

// GRPCClientConfig 封装所有 gRPC 服务的客户端目标地址
type GRPCClientConfig struct {
	ConnectTargetAddr string // Connect 服务的地址
	DeviceTargetAddr  string // Device 服务的地址
	MessageTargetAddr string // Message 服务的地址
	RoomTargetAddr    string // Room 服务的地址
	AuthTargetAddr    string // Auth 服务的地址
}

// AuthEndpoints 封装了Auth服务的监听端点
type AuthEndpoints struct {
	LocalAddr string `yaml:"local_addr"`
	RPCAddr   string `yaml:"rpc_addr"` // gRPC服务监听地址
}

// GatewayEndpoints 封装了Gateway服务的监听端点
type GatewayEndpoints struct {
	Port int `yaml:"port"` // HTTP服务监听端口
}
