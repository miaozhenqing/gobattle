package main

import (
	"context"
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/example/config"
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
		btrouter.Auth(btsession.IsLoggedIn),
		btrouter.SlowLog(50*time.Millisecond),
	)
	mgr := btsession.NewManager()
	r.RegisterModules(&module.PingModule{}, module.NewLoginModule(mgr, r))

	sessionHandler := btsession.NewSessionHandler(mgr)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, sessionHandler)
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

func onPush(msgId uint32, data []byte) {
	btlog.Ins().Infof("push msgId:%v data:%s", msgId, data)
}

func dial() *btrouter.Client {
	cfg := config.ServerConfig
	addr := cfg.Host + ":" + strconv.Itoa(cfg.TcpPort)
	c, err := btrouter.Dial(addr, cfg.MaxPacketSize, btrouter.JSONCodec{}, onPush)
	if err != nil {
		panic(err)
	}
	return c
}

func call[Resp any](c *btrouter.Client, msgId uint32, req any) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := btrouter.Call[Resp](ctx, c, msgId, req)
	btlog.Ins().Infof("call msgId:%v resp:%+v err:%v", msgId, resp, err)
}

func runClient() {
	c1 := dial()
	call[proto.PingResp](c1, proto.MsgPing, &proto.PingReq{Text: "hello"})
	call[proto.LoginResp](c1, proto.MsgLogin, &proto.LoginReq{Token: "10001"})
	call[proto.PingResp](c1, proto.MsgPing, &proto.PingReq{Text: "hello"})

	c2 := dial()
	defer c2.Close()
	call[proto.LoginResp](c2, proto.MsgLogin, &proto.LoginReq{Token: "10001"})

	for {
		call[proto.PingResp](c2, proto.MsgPing, &proto.PingReq{Text: "hello world"})
		time.Sleep(3 * time.Second)
	}
}
