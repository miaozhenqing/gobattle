package btnet

import (
	"fmt"
	"time"
)

type Options struct {
	Name      string
	Host      string
	TcpPort   int
	IPVersion string

	MaxPacketSize  int // 单帧最大长度（含帧头）
	MaxConn        int
	IOReadBuffSize int

	WriteTimeout  time.Duration // 单次写 socket 超时，<=0 表示不设置
	SendQueueSize int           // 每个连接的发送队列长度

	FlushTimeout time.Duration // 优雅关闭时排空发送队列的总超时，<=0 表示不限制
	IdleTimeout  time.Duration // 读空闲超时，期间没收到完整帧即断开，<=0 表示不检测
}

func DefaultOptions() Options {
	return Options{
		Name:           "gobattle",
		Host:           "127.0.0.1",
		TcpPort:        9528,
		IPVersion:      "tcp4",
		MaxPacketSize:  4096,
		MaxConn:        12000,
		IOReadBuffSize: 1024,
		WriteTimeout:   5 * time.Second,
		SendQueueSize:  64,
		FlushTimeout:   3 * time.Second,
		IdleTimeout:    30 * time.Second,
	}
}
func (o *Options) validate() error {
	switch {
	case o.MaxPacketSize < FrameHeaderSize:
		return fmt.Errorf("MaxPacketSize %d < FrameHeaderSize %d", o.MaxPacketSize, FrameHeaderSize)
	case o.IOReadBuffSize <= 0:
		return fmt.Errorf("IOReadBuffSize must be > 0")
	case o.SendQueueSize <= 0:
		return fmt.Errorf("SendQueueSize must be > 0")
	case o.MaxConn <= 0:
		return fmt.Errorf("MaxConn must be > 0")
	}
	return nil
}
