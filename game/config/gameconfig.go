package config

import (
	"gobattle/framework/btconfig"
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
	panic("TODO: load game JSON config and apply flag overrides")
}

func (c *GameConfig) RpcAddr() string {
	panic("TODO: choose advertised RPC host and port")
}
