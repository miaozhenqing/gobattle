package main

import (
	"context"
	"gobattle/common/api"
	"gobattle/common/redis"
	"gobattle/common/redis/sessionstore"
	"gobattle/framework/btconfig"
	"gobattle/framework/btlog"
	"gobattle/framework/btnet"
	"gobattle/framework/btrouter"
	"gobattle/framework/btrpc"
	"gobattle/game/config"
	"gobattle/game/module"
	"gobattle/game/repository"
	"gobattle/game/service"
	"time"
)

func setupLogic(a *app) {
	cfg := a.cfg
	db := openDB(cfg.DB)
	a.onStop(func(ctx context.Context) {
		if err := db.Close(); err != nil {
			btlog.Ins().Error("dbCloseError", "err", err)
		}
	})

	queryTimeout := cfg.DB.QueryTimeout.Duration()
	redisClient := redis.NewClient(cfg.Redis)
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		pingCancel()
		redisClient.Close()
		panic(err)
	}
	pingCancel()
	a.onStop(func(ctx context.Context) { redisClient.Close() })
	dsRouter := btrouter.NewRouterHandler(btrouter.JSONCodec{})
	pushService := service.NewPushService(dsRouter, sessionstore.NewOnlineReader(redisClient))
	userService := service.NewUserService(db, a.idNode, queryTimeout)
	itemService := service.NewItemService(db, pushService, a.idNode, queryTimeout, a.tplMgr)
	shopService := service.NewShopService(db, a.tplMgr, itemService, a.idNode, queryTimeout)

	btrpc.Register[api.UserApi](a.rpcServer, module.NewUserApiModule(userService))
	btrpc.Register[api.ItemApi](a.rpcServer, module.NewItemApiModule(itemService))
	btrpc.Register[api.ShopApi](a.rpcServer, module.NewShopApiModule(shopService))
	btrpc.Register[api.GmApi](a.rpcServer, module.NewGmApiModule(itemService))

	if cfg.DsEnable {
		dsServer := newDsServer(cfg.DsServer, dsRouter, service.NewDsService())
		if err := dsServer.Server.Start(); err != nil {
			panic(err)
		}
		a.onStop(func(ctx context.Context) { dsServer.Shutdown(ctx) })
	}
	btlog.Ins().Info("logicSetup", "queryTimeout", queryTimeout.String())
}

func openDB(c config.DBConfig) *repository.DB {
	openCtx, openCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer openCancel()
	db, err := repository.OpenDB(openCtx, repository.DBOptions{
		Host: c.Host, Port: c.Port, User: c.User, Password: c.Password, Database: c.Database,
		MaxOpenConns: c.MaxOpenConns, MaxIdleConns: c.MaxIdleConns,
		ConnMaxLifetime: c.ConnMaxLifetime.Duration(), ConnMaxIdleTime: c.ConnMaxIdleTime.Duration(),
	})
	if err != nil {
		panic(err)
	}
	return db
}

func newDsServer(c btconfig.ServerConfig, router *btrouter.RouterHandler, dsService *service.DsService) *btnet.ServerGroup {
	router.AddInterceptor(btrouter.SlowLog(50 * time.Millisecond))
	router.RegisterModules(module.NewDsModule(dsService))

	worker := btnet.NewEventLoopGroup("ds-worker", c.IOLoops())
	biz := btnet.NewEventLoopGroup("ds-biz", c.BizLoops())
	s := btnet.NewServer(c.Options(), worker)
	connHandler := module.NewDsConnHandler(dsService)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, connHandler)
		p.AddLastGroup(biz, router)
	})
	return &btnet.ServerGroup{Name: "ds", Server: s, Groups: []*btnet.EventLoopGroup{worker, biz}}
}
