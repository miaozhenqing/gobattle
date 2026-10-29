package config

import (
	"gobattle/btnet"
	"time"
)

// todo 后续从配置文件读取
var DsServerConfig *dsServerConfig

type dsServerConfig struct {
	btnet.Options
}

func init() {
	opts := btnet.DefaultOptions()
	opts.Name = "gobattle-ds"
	opts.TcpPort = 9529
	opts.MaxConn = 1000
	opts.WriteTimeout = time.Second
	DsServerConfig = &dsServerConfig{Options: opts}
}
