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
	"gobattle/game/proto"
	"gobattle/game/repository"
	"gobattle/game/service"
	"os"
	"os/signal"
	"strconv"
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

	//系统相关配置
	boss := btnet.NewEventLoopGroup("boss", 1)
	worker := btnet.NewEventLoopGroup("worker", cfg.IOLoopSize)
	biz := btnet.NewEventLoopGroup("biz", cfg.BizLoopSize)

	//game server
	s := btnet.NewServer(cfg.Options, boss, worker)

	//ds server，这里暂时公用boss和worker
	dsServer := btnet.NewServer(config.DsServerConfig.Options, boss, worker)

	//业务路由handler
	r := btrouter.NewRouterHandler(btrouter.JSONCodec{})
	r.AddInterceptor(
		btrouter.RateLimit(20, 40),
		btrouter.Auth(btsession.IsLoggedIn),
		btrouter.SlowLog(50*time.Millisecond),
	)

	mgr := btsession.NewManager()
	sessionHandler := btsession.NewSessionHandler(mgr)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, sessionHandler)
		p.AddLastGroup(biz, r)
	})

	//业务相关配置
	deps := &service.Deps{DB: repository.NewDB(d)}
	deps.UserService = service.NewUserService(deps)
	deps.ItemService = service.NewItemService(deps)

	moduleDeps := &module.Deps{
		Router:       r,
		SessionMgr:   mgr,
		QueryTimeout: dbCfg.QueryTimeout,
		Deps:         deps,
	}
	moduleDeps.DsModule = module.NewDsModule(moduleDeps)
	matchModule := module.NewBattleModule(moduleDeps)
	r.RegisterModules(
		module.NewLoginModule(moduleDeps),
		moduleDeps.DsModule,
		matchModule,
	)

	dsHandler := module.NewDsHandler(moduleDeps.DsModule)
	dsServer.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, dsHandler)
		p.AddLastGroup(biz, r)
	})

	if err = s.Start(); err != nil {
		panic("server start error")
	}
	if err = dsServer.Start(); err != nil {
		panic("ds server start error")
	}

	//测试代码
	//开始匹配
	matchModule.RunMatch()

	go runClient()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	matchModule.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		btlog.Ins().Error("serverShutdownError", "err", err)
	}
	if err := dsServer.Shutdown(ctx); err != nil {
		btlog.Ins().Error("dsServerShutdownError", "err", err)
	}

	for _, g := range []*btnet.EventLoopGroup{boss, worker, biz} {
		if err := g.ShutdownGracefully(ctx); err != nil {
			btlog.Ins().Error("eventloopShutdownError", "err", err)
		}
	}

	if err := d.Close(); err != nil {
		btlog.Ins().Error("dbCloseError", "err", err)
	}
}

func runClient() {
	client := dial()
	defer client.Close()
	//登录和其他业务
	loginResp, _ := call[proto.LoginResp](client, proto.MsgLogin, &proto.LoginReq{Username: "test1"})

	//心跳
	if config.ServerConfig.IdleTimeout > 0 {
		//需要比IdleTimeout小，否则会触发关闭。写个10测试下
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					call[proto.HeartBeatResp](client, proto.MsgHeartBeat, &proto.HeartBeatReq{UserId: loginResp.UserId})
				case <-client.Done():
					btlog.Ins().Info("clientHeartbeatStop")
					return
				}
			}
		}()
	}

	//模拟匹配
	_, msErr := call[proto.MatchStartResp](client, proto.MsgMatchStart, &proto.MatchStartReq{GameMode: 1, MapID: 1})
	if msErr != nil {
		btlog.Ins().Info("clientMatchStartError", "err", msErr)
	}

	//客户端等待
	<-client.Done()
	btlog.Ins().Info("clientExit")
}

func dial() *btrouter.Client {
	return dialWithPush(config.ServerConfig.Options, onPush)
}

func dialWithPush(cfg btnet.Options, onPush func(msgId uint32, data []byte)) *btrouter.Client {
	addr := cfg.Host + ":" + strconv.Itoa(cfg.TcpPort)
	c, err := btrouter.Dial(addr, cfg.MaxPacketSize, btrouter.JSONCodec{}, onPush)
	if err != nil {
		panic(err)
	}
	return c
}

func onPush(msgId uint32, data []byte) {
	btlog.Ins().Info("clientPush", "msgId", msgId, "data", string(data))
}

func call[Resp any](c *btrouter.Client, msgId uint32, req any) (*Resp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := btrouter.Call[Resp](ctx, c, msgId, req)
	return resp, err
}
