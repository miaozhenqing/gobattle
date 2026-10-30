package config

import (
	"gobattle/btnet"
	"runtime"
	"time"
)

// todo 后续从配置文件读取
var (
	GameServer *ServerConfig
	DsServer   *ServerConfig
)

type ServerConfig struct {
	btnet.Options

	//io loop 数量
	IOLoopSize int
	//业务 loop 数量
	BizLoopSize int
	//节点 id
	NodeId int64
}

func init() {
	GameServer = &ServerConfig{
		Options:     btnet.DefaultOptions(),
		IOLoopSize:  runtime.NumCPU(),
		BizLoopSize: 10,
		NodeId:      1,
	}

	dsOpts := btnet.DefaultOptions()
	dsOpts.Name = "gobattle-ds"
	dsOpts.TcpPort = 9529
	dsOpts.MaxConn = 1000
	dsOpts.WriteTimeout = time.Second
	DsServer = &ServerConfig{
		Options:     dsOpts,
		IOLoopSize:  2,
		BizLoopSize: 2,
	}
}
