package main

import (
	"context"
	"errors"
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
	r.Use(
		btrouter.RateLimit(20, 40),
		btrouter.Auth(func(ctx *btrouter.Context) bool { return true }, router.MsgPing), // 暂无登录协议，先全部放行
		btrouter.SlowLog(50*time.Millisecond),
	)
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
	go func() {
		// 广播：整帧只编码一次
		for range time.Tick(5 * time.Second) {
			if err := r.Broadcast(s.ConnManager(), 50001, &router.PingResp{Text: "broadcast"}); err != nil {
				btlog.Ins().Errorf("broadcast error:%v", err)
			}
		}
	}()
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
		// 限流，连续发 100 次
		limited := 0
		for range 100 {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := btrouter.Call[router.PingResp](ctx, c, router.MsgPing, &router.PingReq{Text: "flood"})
			cancel()
			if e, ok := errors.AsType[*btrouter.Error](err); ok && e.Code == btrouter.CodeResourceExhausted {
				limited++
			}
		}
		btlog.Ins().Infof("flood 100, limited:%v", limited)
		time.Sleep(3 * time.Second)
	}
}
