package btconfig

import (
	"bytes"
	"encoding/json"
	"gobattle/framework/btnet"
	"os"
	"runtime"
	"time"
)

type Duration time.Duration

func (d Duration) Duration() time.Duration { return time.Duration(d) }

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(v)
	return nil
}

func LoadJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

type ServerConfig struct {
	Name          string    `json:"name"`
	Host          string    `json:"host"`
	TcpPort       int       `json:"tcpPort"`
	MaxPacketSize int       `json:"maxPacketSize"`
	MaxConn       int       `json:"maxConn"`
	WriteTimeout  Duration  `json:"writeTimeout"`
	SendQueueSize int       `json:"sendQueueSize"`
	IdleTimeout   *Duration `json:"idleTimeout"`
	IOLoopSize    int       `json:"ioLoopSize"`
	BizLoopSize   int       `json:"bizLoopSize"`
}

func (c *ServerConfig) Options() btnet.Options {
	o := btnet.DefaultOptions()
	if c.Name != "" {
		o.Name = c.Name
	}
	if c.Host != "" {
		o.Host = c.Host
	}
	if c.TcpPort > 0 {
		o.TcpPort = c.TcpPort
	}
	if c.MaxPacketSize > 0 {
		o.MaxPacketSize = c.MaxPacketSize
	}
	if c.MaxConn > 0 {
		o.MaxConn = c.MaxConn
	}
	if c.WriteTimeout > 0 {
		o.WriteTimeout = c.WriteTimeout.Duration()
	}
	if c.SendQueueSize > 0 {
		o.SendQueueSize = c.SendQueueSize
	}
	if c.IdleTimeout != nil {
		o.IdleTimeout = c.IdleTimeout.Duration()
	}
	return o
}

func (c *ServerConfig) IOLoops() int {
	if c.IOLoopSize <= 0 {
		return runtime.NumCPU()
	}
	return c.IOLoopSize
}

func (c *ServerConfig) BizLoops() int {
	if c.BizLoopSize <= 0 {
		return 1
	}
	return c.BizLoopSize
}

type ZkConfig struct {
	Addrs          []string `json:"addrs"`
	Root           string   `json:"root"`
	SessionTimeout Duration `json:"sessionTimeout"`
}
