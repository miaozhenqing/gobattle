package main

import (
	"context"
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/config"
	"gobattle/game/db"
	"gobattle/game/module"
	"gobattle/game/service"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.ServerConfig
	dbCfg := config.DBConfig

	openCtx, openCancel := context.WithTimeout(context.Background(), 5*time.Second)
	d, err := db.Open(openCtx, db.Options{
		Host:            dbCfg.Host,
		Port:            dbCfg.Port,
		User:            dbCfg.User,
		Password:        dbCfg.Password,
		Database:        dbCfg.Database,
		MaxOpenConns:    dbCfg.MaxOpenConns,
		MaxIdleConns:    dbCfg.MaxIdleConns,
		ConnMaxLifetime: dbCfg.ConnMaxLifetime,
		ConnMaxIdleTime: dbCfg.ConnMaxIdleTime,
	})
	openCancel()
	if err != nil {
		panic(err)
	}

	accountService := service.NewAccountService(d)

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
	deps := &module.Deps{
		Router:         r,
		SessionMgr:     mgr,
		QueryTimeout:   dbCfg.QueryTimeout,
		AccountService: accountService,
	}
	r.RegisterModules(
		module.NewLoginModule(deps),
	)

	sessionHandler := btsession.NewSessionHandler(mgr)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, sessionHandler)
		p.AddLastGroup(biz, r)
	})

	if err = s.Start(); err != nil {
		panic("server start error")
	}

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

	// 业务 loop 停完再关库，避免在途请求拿不到连接
	if err := d.Close(); err != nil {
		btlog.Ins().Errorf("db close error:%v", err)
	}
}
