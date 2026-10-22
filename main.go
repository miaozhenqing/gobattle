package main

import (
	"context"
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/btrouter"
	"gobattle/config"
	"gobattle/handler"
	"gobattle/router"
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

	r := btrouter.NewRouter(btrouter.JSONCodec{})
	(&router.PingModule{}).Routes(r)

	s.SetPipelineInitializer(func(p btiface.IPipeline) {
		p.AddLast(btnet.NewEnvelopeCodec())
		p.AddLastGroup(biz, handler.NewSessionHandler())
		p.AddLastGroup(biz, btnet.NewDispatchHandler(r))
	})

	err := s.Start()
	if err != nil {
		panic("server start error")
	}

	go runClient()
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

func runClient() {
	cfg := config.ServerConfig
	addr := cfg.Host + ":" + strconv.Itoa(cfg.TcpPort)
	c, err := btrouter.Dial(addr, cfg.MaxPacketSize, btrouter.JSONCodec{}, func(msgId uint32, data []byte) {
		btlog.Ins().Infof("push msgId:%v data:%s", msgId, data)
	})
	if err != nil {
		btlog.Ins().Errorf("dial error:%v", err)
		return
	}
	defer c.Close()

	call := func(msgId uint32, req any) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		resp, err := btrouter.Call[router.PingResp](ctx, c, msgId, req)
		btlog.Ins().Infof("call msgId:%v resp:%+v err:%v", msgId, resp, err)
	}
	for {
		call(router.MsgPing, &router.PingReq{Text: "hello world"})
		call(router.MsgPing, &router.PingReq{})
		call(9999, nil)
		time.Sleep(3 * time.Second)
	}
}
