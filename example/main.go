package main

import (
	"context"
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/btrouter"
	"gobattle/example/config"
	"gobattle/example/handler"
	"gobattle/example/module"
	"gobattle/example/proto"
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
	s := btnet.NewServer(cfg.Options, boss, worker)

	r := btrouter.NewRouter(btrouter.JSONCodec{})
	r.AddInterceptor(
		btrouter.RateLimit(20, 40),
		btrouter.Auth(func(ctx *btrouter.RequestContext) bool { return true }), // 暂无登录协议，先全部放行
		btrouter.SlowLog(50*time.Millisecond),
	)
	r.RegisterModules(&module.PingModule{})

	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, handler.NewSessionHandler())
		p.AddLastGroup(biz, r)
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

	for _, g := range []*btnet.EventLoopGroup{boss, worker, biz} {
		if err := g.ShutdownGracefully(ctx); err != nil {
			btlog.Ins().Errorf("eventloop shutdown error:%v", err)
		}
	}

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
		resp, err := btrouter.Call[proto.PingResp](ctx, c, msgId, req)
		btlog.Ins().Infof("call msgId:%v resp:%+v err:%v", msgId, resp, err)
	}
	for {
		call(proto.MsgPing, &proto.PingReq{Text: "hello world"})
		time.Sleep(3 * time.Second)
	}
}
