package config

import (
	"gobattle/btnet"
	"runtime"
	"time"
)

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
	opts.WriteTimeout = time.Second
	ServerConfig = &serverConfig{
		Options:     opts,
		IOLoopSize:  runtime.NumCPU(),
		BizLoopSize: 5,
	}
}
