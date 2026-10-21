package main

import (
	"context"
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/config"
	"gobattle/handler"
	"gobattle/router"
	"gobattle/util"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	cfg := config.ServerConfig
	boss := btnet.NewEventLoopGroup("boss", 1)
	worker := btnet.NewEventLoopGroup("worker", cfg.IOLoopSize)
	biz := btnet.NewEventLoopGroup("biz", cfg.BizLoopSize)
	s := btnet.NewServer(serverOptions(), boss, worker)

	gdh := handler.NewGameDispatcher()
	gdh.AddRouter(util.MessageIDPing, router.NewPingRouter())

	s.SetPipelineInitializer(func(p btiface.IPipeline) {
		p.AddLastGroup(biz, handler.NewSessionHandler())
		p.AddLastGroup(biz, btnet.NewDispatchHandler(gdh))
		p.AddLast(btnet.NewEncodeHandler(btnet.NewSerializer(cfg.MaxPacketSize)))
	})

	err := s.Start()
	if err != nil {
		panic("server start error")
	}

	go NewClient().Test()
	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		btlog.Ins().Errorf("server shutdown error:%v", err)
	}
	// 连接全部结束后再关 loop，顺序：boss（accept 已退出）→ worker → biz
	for _, g := range []*btnet.EventLoopGroup{boss, worker, biz} {
		if err := g.ShutdownGracefully(ctx); err != nil {
			btlog.Ins().Errorf("eventloop shutdown error:%v", err)
		}
	}

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
	opts.IOReadBuffSize = cfg.IOReadBuffSize
	opts.WriteTimeout = cfg.WriteTimeout
	opts.SendQueueSize = cfg.SendQueueSize
	opts.FlushTimeout = cfg.FlushTimeout
	opts.IdleTimeout = cfg.IdleTimeout
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
