package config

import (
    "time"
)

// Config 应用程序配置
type Config struct {
    Server   ServerConfig   `yaml:"server" json:"server"`
    RPC      RPCConfig      `yaml:"rpc" json:"rpc"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
    Name         string        `yaml:"name" json:"name"`
    Port         int           `yaml:"port" json:"port"`
    ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
    WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
}

// RPCConfig RPC 服务配置
type RPCConfig struct {
    // 当前服务作为 gRPC 服务端时的配置
    Server RPCServerConfig `yaml:"server" json:"server"`
    // 当前服务作为 gRPC 客户端时的配置
    Clients map[string]RPCClientConfig `yaml:"clients" json:"clients"`
}

// RPCServerConfig RPC 服务端配置
type RPCServerConfig struct {
    Port    int           `yaml:"port" json:"port"`
    Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// RPCClientConfig RPC 客户端配置
type RPCClientConfig struct {
    Address     string        `yaml:"address" json:"address"`
    Timeout     time.Duration `yaml:"timeout" json:"timeout"`
    MaxRetries  int           `yaml:"max_retries" json:"max_retries"`
    KeepAlive   time.Duration `yaml:"keep_alive" json:"keep_alive"`
    DialTimeout time.Duration `yaml:"dial_timeout" json:"dial_timeout"`
}
