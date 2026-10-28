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
	s := btnet.NewServer(cfg.Options, boss, worker)

	r := btrouter.NewRouter(btrouter.JSONCodec{})
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
	matchModule := module.NewMatchModule(moduleDeps)
	r.RegisterModules(
		module.NewLoginModule(moduleDeps),
		moduleDeps.DsModule,
		matchModule,
	)

	if err = s.Start(); err != nil {
		panic("server start error")
	}

	//测试代码
	//开始匹配
	matchModule.Run()
	go runClient()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	matchModule.Stop()

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

	if err := d.Close(); err != nil {
		btlog.Ins().Errorf("db close error:%v", err)
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
					call[proto.HeartBeatResp](client, proto.MsgHeartBeat, &proto.HeartBeatReq{UID: loginResp.UID})
				case <-client.Done():
					btlog.Ins().Infof("client done, stop heartbeat")
					return
				}
			}
		}()
	}

	//模拟匹配
	_, msErr := call[proto.MatchStartResp](client, proto.MsgMatchStart, &proto.MatchStartReq{GameMode: 1, MapID: 1})
	if msErr != nil {
		btlog.Ins().Infof(">>>>>match start error:%v", msErr)
	}

	//客户端等待
	<-client.Done()
	btlog.Ins().Infof("client done, exit runClient")
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

func onPush(msgId uint32, data []byte) {
	btlog.Ins().Infof("push msgId:%v data:%s", msgId, data)
}

func call[Resp any](c *btrouter.Client, msgId uint32, req any) (*Resp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := btrouter.Call[Resp](ctx, c, msgId, req)
	return resp, err
}
