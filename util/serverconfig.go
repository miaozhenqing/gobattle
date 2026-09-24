package utils

import (
	"os"
)

var ServerConfig *serverConfig

type serverConfig struct {
	Host      string
	TcpPort   int
	Name      string
	IPVersion string

	MaxPacketSize    uint32
	MaxConn          uint32
	WorkerPoolSize   uint32
	MaxWorkerTaskLen uint32
	IOReadBuffSize   uint32
}

func init() {
	ServerConfig = &serverConfig{
		Name:             "gobattle",
		IPVersion:        "tcp4",
		TcpPort:          9528,
		Host:             "127.0.0.1",
		MaxConn:          12000,
		MaxPacketSize:    4096,
		WorkerPoolSize:   5,
		MaxWorkerTaskLen: 1024,
		IOReadBuffSize:   1024,
	}
	ServerConfig.Reload()
}
func (g *serverConfig) Reload() {
	_, _ = os.ReadFile("conf/global.json")
	//todo
}
