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

	"github.com/jmoiron/sqlx"
)

// 方便统一关闭
type serverWrapper struct {
	name   string
	server *btnet.Server
	groups []*btnet.EventLoopGroup
}

func (b *serverWrapper) shutdown(ctx context.Context) {
	if err := b.server.Shutdown(ctx); err != nil {
		btlog.Ins().Error("serverShutdownError", "server", b.name, "err", err)
	}
	for _, g := range b.groups {
		if err := g.ShutdownGracefully(ctx); err != nil {
			btlog.Ins().Error("eventloopShutdownError", "server", b.name, "err", err)
		}
	}
}

func main() {
	d := openDB()
	deps := newModuleDeps(d)

	dsModule := module.NewDsModule(deps)
	loginModule := module.NewLoginModule(deps)
	battleModule := module.NewBattleModule(deps, dsModule)
	deps.GameRouterHandler.RegisterModules(loginModule, battleModule)
	deps.DsRouterHandler.RegisterModules(dsModule)
	battleModule.RegisterDsRoutes(deps.DsRouterHandler)

	gameServer := newGameServer(deps)
	dsServer := newDsServer(deps, dsModule)

	if err := dsServer.server.Start(); err != nil {
		panic("ds server start error")
	}
	if err := gameServer.server.Start(); err != nil {
		panic("game server start error")
	}

	//开始匹配
	battleModule.RunMatch()

	//测试代码
	go runClient()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	battleModule.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gameServer.shutdown(ctx)
	dsServer.shutdown(ctx)

	if err := d.Close(); err != nil {
		btlog.Ins().Error("dbCloseError", "err", err)
	}
}

func openDB() *sqlx.DB {
	dbCfg := config.DBConfig
	openCtx, openCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer openCancel()
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
	if err != nil {
		panic(err)
	}
	return d
}

func newModuleDeps(d *sqlx.DB) *module.Deps {
	serviceDeps := &service.Deps{DB: repository.NewDB(d)}
	serviceDeps.UserService = service.NewUserService(serviceDeps)
	serviceDeps.ItemService = service.NewItemService(serviceDeps)

	gameRouter := btrouter.NewRouterHandler(btrouter.JSONCodec{})
	gameRouter.AddInterceptor(
		btrouter.RateLimit(20, 40),
		btrouter.Auth(btsession.IsLoggedIn),
		btrouter.SlowLog(50*time.Millisecond),
	)

	// ds为内部服务，不限流
	dsRouter := btrouter.NewRouterHandler(btrouter.JSONCodec{})
	dsRouter.AddInterceptor(
		//btrouter.Auth(module.IsDsRegistered),
		btrouter.SlowLog(50 * time.Millisecond),
	)

	return &module.Deps{
		GameRouterHandler: gameRouter,
		DsRouterHandler:   dsRouter,
		SessionMgr:        btsession.NewManager(),
		QueryTimeout:      config.DBConfig.QueryTimeout,
		UserService:       serviceDeps.UserService,
		ItemService:       serviceDeps.ItemService,
	}
}

func newGameServer(deps *module.Deps) *serverWrapper {
	cfg := config.ServerConfig
	boss := btnet.NewEventLoopGroup("game-boss", 1)
	worker := btnet.NewEventLoopGroup("game-worker", cfg.IOLoopSize)
	biz := btnet.NewEventLoopGroup("game-biz", cfg.BizLoopSize)

	s := btnet.NewServer(cfg.Options, boss, worker)
	sessionHandler := btsession.NewSessionHandler(deps.SessionMgr)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, sessionHandler)
		p.AddLastGroup(biz, deps.GameRouterHandler)
	})
	return &serverWrapper{name: "game", server: s, groups: []*btnet.EventLoopGroup{boss, worker, biz}}
}

func newDsServer(deps *module.Deps, dsModule *module.DsModule) *serverWrapper {
	cfg := config.DsServerConfig
	boss := btnet.NewEventLoopGroup("ds-boss", 1)
	worker := btnet.NewEventLoopGroup("ds-worker", cfg.IOLoopSize)
	biz := btnet.NewEventLoopGroup("ds-biz", cfg.BizLoopSize)

	s := btnet.NewServer(cfg.Options, boss, worker)
	sessionHandler := module.NewDsSessionHandler(dsModule)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, sessionHandler)
		p.AddLastGroup(biz, deps.DsRouterHandler)
	})
	return &serverWrapper{name: "ds", server: s, groups: []*btnet.EventLoopGroup{boss, worker, biz}}
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
