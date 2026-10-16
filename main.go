package main

import (
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/config"
	"gobattle/handler"
	"gobattle/router"
	"gobattle/util"
	"net"
	"strconv"
	"time"
)

func main() {
	s := btnet.NewServer(serverOptions())

	serializer := btnet.NewSerializer(config.ServerConfig.MaxPacketSize)

	gdh := handler.NewGameDispatcher()
	gdh.AddRouter(util.MessageIDPing, router.NewPingRouter())

	s.SetPipelineInitializer(func(p btiface.IPipeline) {
		p.AddLast(btnet.NewDispatchHandler(gdh))
		p.AddLast(btnet.NewEncodeHandler(serializer))
	})

	err := s.Start()
	if err != nil {
		panic("server start error")
	}
	client := NewClient()
	client.Test()
}

func serverOptions() btnet.Options {
	cfg := config.ServerConfig
	opts := btnet.DefaultOptions()
	opts.Name = cfg.Name
	opts.Host = cfg.Host
	opts.TcpPort = cfg.TcpPort
	opts.IPVersion = cfg.IPVersion
	opts.MaxPacketSize = cfg.MaxPacketSize
	opts.MaxConn = cfg.MaxConn
	opts.WorkerPoolSize = cfg.WorkerPoolSize
	opts.WorkerPoolQueueSize = cfg.WorkerPoolQueueSize
	opts.IOReadBuffSize = cfg.IOReadBuffSize
	opts.WriteTimeout = cfg.WriteTimeout
	opts.SendQueueSize = cfg.SendQueueSize
	return opts
}

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}
func (c *Client) Test() {
	resolveTCPAddr, _ := net.ResolveTCPAddr("tcp", config.ServerConfig.Host+":"+strconv.Itoa(config.ServerConfig.TcpPort))

	conn, err := net.DialTCP("tcp", nil, resolveTCPAddr)
	if err != nil {
		panic(err)
	}
	serializer := btnet.NewSerializer(config.ServerConfig.MaxPacketSize)
	for {
		s := "hello world"
		message := btnet.NewMessage(util.MessageIDPing, []byte(s))

		buf := serializer.Serialize(message)

		conn.Write(buf)

		// 接收返回值
		response := make([]byte, 1024)

		n, err := conn.Read(response)
		if err != nil {
			btlog.Ins().Errorf("read error:%v", err)
			return
		}
		deserialize, err := serializer.Deserialize(response[:n])
		btlog.Ins().Info("receive:", string(deserialize.GetData()))

		time.Sleep(time.Second * 3)
	}

}
