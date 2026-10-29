package config

import "time"

// todo 后续从配置文件读取
var DB *DBConfig

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration // 连接最大生命周期
	ConnMaxIdleTime time.Duration // 连接最大空闲时间

	// 单次业务请求内 DB 操作的总超时
	QueryTimeout time.Duration
}

func init() {
	DB = &DBConfig{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Password: "admin",
		Database: "gobattle",

		MaxOpenConns:    20,
		MaxIdleConns:    10,
		ConnMaxLifetime: 0 * time.Hour,
		ConnMaxIdleTime: 30 * time.Minute,

		QueryTimeout: 3 * time.Second,
	}
}
