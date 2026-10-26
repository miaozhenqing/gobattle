package config

import "time"

var DBConfig *dbConfig

type dbConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	// 单次业务请求内 DB 操作的总超时
	QueryTimeout time.Duration
}

func init() {
	DBConfig = &dbConfig{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Password: "root",
		Database: "gobattle",

		MaxOpenConns:    20,
		MaxIdleConns:    10,
		ConnMaxLifetime: time.Hour,
		ConnMaxIdleTime: 10 * time.Minute,

		QueryTimeout: 3 * time.Second,
	}
}
