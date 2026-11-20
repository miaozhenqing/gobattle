package config

import (
	_ "encoding/json"
	"flag"
	"fmt"
	"gobattle/framework/btconfig"
	_ "os"
	"strconv"
	"time"
)

var Game *GameConfig

type GameConfig struct {
	NodeId           int64                 `json:"nodeId"`
	Role             string                `json:"role"`
	AdvertiseHost    string                `json:"advertiseHost"`
	TplDir           string                `json:"tplDir"`
	Rpc              btconfig.ServerConfig `json:"rpc"`
	CallTimeout      btconfig.Duration     `json:"callTimeout"`
	RpcMaxPacketSize int                   `json:"rpcMaxPacketSize"`
	DsEnable         bool                  `json:"dsEnable"`
	DsServer         btconfig.ServerConfig `json:"dsServer"`
	Zk               btconfig.ZkConfig     `json:"zk"`
	DB               DBConfig              `json:"db"`
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
	path := flag.String("config", "game/game.json", "config file")
	role := flag.String("role", "", "override role")
	nodeId := flag.Int64("nodeId", 0, "override nodeId")
	port := flag.Int("port", 0, "override rpc.tcpPort")
	advertiseHost := flag.String("advertiseHost", "", "override advertiseHost")
	flag.Parse()

	cfg := &GameConfig{TplDir: "game/tpl/json", CallTimeout: btconfig.Duration(3 * time.Second), RpcMaxPacketSize: 65536}
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

	if *role != "" {
		cfg.Role = *role
	}
	if cfg.NodeId <= 0 || cfg.Role == "" {
		panic("nodeId and role are required")
	}
	Game = cfg
}

func (c *GameConfig) RpcAddr() string {
	host := c.AdvertiseHost
	if host == "" {
		host = c.Rpc.Host
	}
	return host + ":" + strconv.Itoa(c.Rpc.TcpPort)
}
