package config

import (
	"encoding/json"
	"gobattle/framework/btconfig"
	"os"
)

var Gateway *GatewayConfig

type GatewayConfig struct {
	NodeId           int64                 `json:"nodeId"`
	Server           btconfig.ServerConfig `json:"server"`
	Zk               btconfig.ZkConfig     `json:"zk"`
	CallTimeout      btconfig.Duration     `json:"callTimeout"`
	RpcMaxPacketSize int                   `json:"rpcMaxPacketSize"`
}

func Load() {
	//todo 先采用相对于可执行文件目录
	file, err := os.ReadFile("gateway/config/gateway.json")
	if err != nil {
		panic(err)
	}
	var cfg GatewayConfig
	err = json.Unmarshal(file, &cfg)
	if err != nil {
		panic(err)
	}
	Gateway = &cfg
}
