package redis

import (
	"time"

	redisclient "github.com/redis/go-redis/v9"
)

type Config struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

func NewClient(cfg Config) *redisclient.Client {
	return redisclient.NewClient(&redisclient.Options{
		Addr:                  cfg.Addr,
		Password:              cfg.Password,
		DB:                    cfg.DB,
		DialTimeout:           2 * time.Second,
		ReadTimeout:           2 * time.Second,
		WriteTimeout:          2 * time.Second,
		ContextTimeoutEnabled: true,
	})
}
