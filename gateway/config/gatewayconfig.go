package config

import (
	"flag"
	"fmt"
	"gobattle/framework/btconfig"
	"time"
)

var Gateway *GatewayConfig

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type GatewayConfig struct {
	NodeId            int64                 `json:"nodeId"`
	Server            btconfig.ServerConfig `json:"server"`
	Zk                btconfig.ZkConfig     `json:"zk"`
	Redis             RedisConfig           `json:"redis"`
	SessionTTL        btconfig.Duration     `json:"sessionTTL"`
	ReconcileInterval btconfig.Duration     `json:"reconcileInterval"`
	CallTimeout       btconfig.Duration     `json:"callTimeout"`
	RpcMaxPacketSize  int                   `json:"rpcMaxPacketSize"`
}

func Load() {
	path := flag.String("config", "gateway/gateway.json", "config file")
	nodeId := flag.Int64("nodeId", 0, "override nodeId")
	port := flag.Int("port", 0, "override server.tcpPort")
	flag.Parse()

	cfg := &GatewayConfig{
		CallTimeout:       btconfig.Duration(3 * time.Second),
		RpcMaxPacketSize:  65536,
		SessionTTL:        btconfig.Duration(90 * time.Second),
		ReconcileInterval: btconfig.Duration(20 * time.Second),
	}
	if err := btconfig.LoadJSON(*path, cfg); err != nil {
		panic(fmt.Sprintf("load config %s: %v", *path, err))
	}
	if *nodeId > 0 {
		cfg.NodeId = *nodeId
	}
	if *port > 0 {
		cfg.Server.TcpPort = *port
	}
	if cfg.NodeId <= 0 {
		panic("nodeId is required")
	}
	if cfg.Redis.Addr == "" || cfg.SessionTTL <= 2*cfg.ReconcileInterval || cfg.ReconcileInterval <= 0 {
		panic("invalid redis/sessionTTL/reconcileInterval")
	}
	Gateway = cfg
}
