package config

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
	}
}
