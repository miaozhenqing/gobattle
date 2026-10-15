package btnet

import "time"

type Options struct {
	Name      string
	Host      string
	TcpPort   int
	IPVersion string

	MaxPacketSize       uint32 // 单帧最大长度（含帧头）
	MaxConn             uint32
	WorkerPoolSize      uint32
	WorkerPoolQueueSize uint32
	IOReadBuffSize      uint32

	WriteTimeout  time.Duration // 单次写 socket 超时，<=0 表示不设置
	SendQueueSize uint32        // 每个连接的发送队列长度
}

func DefaultOptions() Options {
	return Options{
		Name:                "gobattle",
		Host:                "127.0.0.1",
		TcpPort:             9528,
		IPVersion:           "tcp4",
		MaxPacketSize:       4096,
		MaxConn:             12000,
		WorkerPoolSize:      5,
		WorkerPoolQueueSize: 1024,
		IOReadBuffSize:      1024,
		WriteTimeout:        5 * time.Second,
		SendQueueSize:       64,
	}
}
