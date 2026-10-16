package config

import "time"

var ServerConfig *serverConfig

type serverConfig struct {
	Host      string
	TcpPort   int
	Name      string
	IPVersion string

	MaxPacketSize       int
	MaxConn             int
	WorkerPoolSize      int
	WorkerPoolQueueSize int
	IOReadBuffSize      int

	//写超时
	WriteTimeout time.Duration
	//发送队列长度
	SendQueueSize int
	//优雅关闭时排空发送队列的总超时
	FlushTimeout time.Duration
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
		FlushTimeout:        3 * time.Second,
	}
}
