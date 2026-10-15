package config

import "time"

var ServerConfig *serverConfig

type serverConfig struct {
	Host      string
	TcpPort   int
	Name      string
	IPVersion string

	MaxPacketSize       uint32
	MaxConn             uint32
	WorkerPoolSize      uint32
	WorkerPoolQueueSize uint32
	IOReadBuffSize      uint32

	//写超时
	WriteTimeout time.Duration
	//发送队列长度
	SendQueueSize uint32
}

func init() {
	ServerConfig = &serverConfig{
		Name:                "gobattle",
		IPVersion:           "tcp4",
		TcpPort:             9528,
		Host:                "127.0.0.1",
		MaxConn:             12000,
		MaxPacketSize:       4096,
		WorkerPoolSize:      5,
		WorkerPoolQueueSize: 1024,
		IOReadBuffSize:      1024,
		WriteTimeout:        time.Second,
		SendQueueSize:       64,
	}
}
