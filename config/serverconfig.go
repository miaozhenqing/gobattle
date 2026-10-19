package config

import (
	"runtime"
	"time"
)

var ServerConfig *serverConfig

type serverConfig struct {
	Host      string
	TcpPort   int
	Name      string
	IPVersion string

	MaxPacketSize  int
	MaxConn        int
	IOReadBuffSize int

	//io loop 数量
	IOLoopSize int
	//业务 loop 数量
	BizLoopSize int

	//写超时
	WriteTimeout time.Duration
	//发送队列长度
	SendQueueSize int
	//优雅关闭时排空发送队列的总超时
	FlushTimeout time.Duration
	//读空闲超时
	IdleTimeout time.Duration
}

func init() {
	ServerConfig = &serverConfig{
		Name:           "gobattle",
		IPVersion:      "tcp4",
		TcpPort:        9528,
		Host:           "127.0.0.1",
		MaxConn:        12000,
		MaxPacketSize:  4096,
		IOLoopSize:     runtime.NumCPU(),
		BizLoopSize:    5,
		IOReadBuffSize: 1024,
		WriteTimeout:   time.Second,
		SendQueueSize:  64,
		FlushTimeout:   3 * time.Second,
		IdleTimeout:    30 * time.Second,
	}
}
