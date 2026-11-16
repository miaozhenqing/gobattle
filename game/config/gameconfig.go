package config

import (
	_ "encoding/json"
	"flag"
	"fmt"
	"gobattle/framework/btconfig"
	_ "os"
)

var Game *GameConfig

type GameConfig struct {
	NodeId        int64                 `json:"nodeId"`
	Group         string                `json:"group"`
	AdvertiseHost string                `json:"advertiseHost"`
	TplDir        string                `json:"tplDir"`
	Rpc           btconfig.ServerConfig `json:"rpc"`
	DsEnable      bool                  `json:"dsEnable"`
	DsServer      btconfig.ServerConfig `json:"dsServer"`
	Zk            btconfig.ZkConfig     `json:"zk"`
	DB            DBConfig              `json:"db"`
}

type DBConfig struct {
	Host            string            `json:"host"`
	Port            int               `json:"port"`
	User            string            `json:"user"`
	Password        string            `json:"password"`
	Database        string            `json:"database"`
	MaxOpenConns    int               `json:"maxOpenConns"`
	MaxIdleConns    int               `json:"maxIdleConns"`
	ConnMaxLifetime btconfig.Duration `json:"connMaxLifetime"`
	ConnMaxIdleTime btconfig.Duration `json:"connMaxIdleTime"`
	QueryTimeout    btconfig.Duration `json:"queryTimeout"`
}

func Load() {
	path := flag.String("config", "game/config/game.json", "config file")
	nodeId := flag.Int64("nodeId", 0, "override nodeId")
	port := flag.Int("port", 0, "override rpc.tcpPort")
	advertiseHost := flag.String("advertiseHost", "", "override advertiseHost")
	flag.Parse()

	cfg := &GameConfig{TplDir: "game/tpl/json"}
	if err := btconfig.LoadJSON(*path, cfg); err != nil {
		panic(fmt.Sprintf("load config %s: %v", *path, err))
	}
	if *nodeId > 0 {
		cfg.NodeId = *nodeId
	}
	if *port > 0 {
		cfg.Rpc.TcpPort = *port
	}
	if *advertiseHost != "" {
		cfg.AdvertiseHost = *advertiseHost
	}
	if cfg.NodeId <= 0 || cfg.Group == "" {
		panic("nodeId and group are required")
	}
	Game = cfg
}

func (c *GameConfig) RpcAddr() string {
	panic("TODO: choose advertised RPC host and port")
}
