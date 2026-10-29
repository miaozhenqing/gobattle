package config

import (
	"gobattle/btnet"
	"runtime"
)

// todo 后续从配置文件读取
var ServerConfig *serverConfig

type serverConfig struct {
	btnet.Options

	//io loop 数量
	IOLoopSize int
	//业务 loop 数量
	BizLoopSize int
}

func init() {
	opts := btnet.DefaultOptions()
	ServerConfig = &serverConfig{
		Options:     opts,
		IOLoopSize:  runtime.NumCPU(),
		BizLoopSize: 10,
	}
}
