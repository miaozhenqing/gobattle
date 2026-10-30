package main

import (
	"context"
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/btrouter"
	"gobattle/btsession"
	"gobattle/game/config"
	"gobattle/game/module"
	"gobattle/game/protocol"
	"gobattle/game/repository"
	"gobattle/game/service"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/snowflake"
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
	db := openDB()
	gameRouterHandler, dsRouterHandler := newRouterHandlers()
	sessionMgr := btsession.NewManager()
	idNode, err := snowflake.NewNode(config.GameServer.NodeId)
	if err != nil {
		panic(err)
	}

	// service：按依赖顺序构造
	userService := service.NewUserService(db, idNode, config.DB.QueryTimeout)
	pushService := service.NewPushService(gameRouterHandler, dsRouterHandler, sessionMgr)
	itemService := service.NewItemService(db, pushService, idNode, config.DB.QueryTimeout)
	dsService := service.NewDsService()
	battleService := service.NewBattleService(dsService, pushService)
	matchService := service.NewMatchService(battleService, pushService, sessionMgr)

	// module
	loginModule := module.NewLoginModule(userService, pushService, sessionMgr)
	battleModule := module.NewBattleModule(matchService, battleService)
	dsModule := module.NewDsModule(dsService)
	itemModule := module.NewItemModule(itemService)
	gameRouterHandler.RegisterModules(loginModule, battleModule, itemModule)
	dsRouterHandler.RegisterModules(dsModule)
	battleModule.RegisterDsRoutes(dsRouterHandler)

	gameServer := newGameServer(gameRouterHandler, sessionMgr)
	dsServer := newDsServer(dsRouterHandler, dsService)

	if err := dsServer.server.Start(); err != nil {
		panic("ds server start error")
	}
	if err := gameServer.server.Start(); err != nil {
		panic("game server start error")
	}

	battleService.Run()
	matchService.Run()

	//测试代码
	go runClient()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()

	// 先停匹配，不再产生新战斗
	matchService.Stop()
	battleService.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gameServer.shutdown(ctx)
	dsServer.shutdown(ctx)

	if err := db.Close(); err != nil {
		btlog.Ins().Error("dbCloseError", "err", err)
	}
}

func openDB() *repository.DB {
	dbCfg := config.DB
	openCtx, openCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer openCancel()
	db, err := repository.OpenDB(openCtx, repository.DBOptions{
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
	return db
}

func newRouterHandlers() (*btrouter.RouterHandler, *btrouter.RouterHandler) {
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

	return gameRouter, dsRouter
}

func newGameServer(gameRouterHandler *btrouter.RouterHandler, sessionMgr *btsession.Manager) *serverWrapper {
	cfg := config.GameServer
	boss := btnet.NewEventLoopGroup("game-boss", 1)
	worker := btnet.NewEventLoopGroup("game-worker", cfg.IOLoopSize)
	biz := btnet.NewEventLoopGroup("game-biz", cfg.BizLoopSize)

	s := btnet.NewServer(cfg.Options, boss, worker)
	sessionHandler := btsession.NewSessionHandler(sessionMgr)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, sessionHandler)
		p.AddLastGroup(biz, gameRouterHandler)
	})
	return &serverWrapper{name: "game", server: s, groups: []*btnet.EventLoopGroup{boss, worker, biz}}
}

func newDsServer(dsRouterHandler *btrouter.RouterHandler, dsService *service.DsService) *serverWrapper {
	cfg := config.DsServer
	boss := btnet.NewEventLoopGroup("ds-boss", 1)
	worker := btnet.NewEventLoopGroup("ds-worker", cfg.IOLoopSize)
	biz := btnet.NewEventLoopGroup("ds-biz", cfg.BizLoopSize)

	s := btnet.NewServer(cfg.Options, boss, worker)
	connHandler := module.NewDsConnHandler(dsService)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, connHandler)
		p.AddLastGroup(biz, dsRouterHandler)
	})
	return &serverWrapper{name: "ds", server: s, groups: []*btnet.EventLoopGroup{boss, worker, biz}}
}

func runClient() {
	client := dial()
	defer client.Close()
	//登录和其他业务
	loginResp, _ := call[protocol.LoginResp](client, protocol.MsgLogin, &protocol.LoginReq{Username: "test1"})

	//心跳
	if config.GameServer.IdleTimeout > 0 {
		//需要比IdleTimeout小，否则会触发关闭。写个10测试下
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					call[protocol.HeartBeatResp](client, protocol.MsgHeartBeat, &protocol.HeartBeatReq{UserId: loginResp.UserId})
				case <-client.Done():
					btlog.Ins().Info("clientHeartbeatStop")
					return
				}
			}
		}()
	}

	//模拟匹配
	_, msErr := call[protocol.MatchStartResp](client, protocol.MsgMatchStart, &protocol.MatchStartReq{GameMode: 1, MapId: 1})
	if msErr != nil {
		btlog.Ins().Info("clientMatchStartError", "err", msErr)
	}

	//客户端等待
	<-client.Done()
	btlog.Ins().Info("clientExit")
}

func dial() *btrouter.Client {
	return dialWithPush(config.GameServer.Options, onPush)
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
