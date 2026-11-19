package main

import (
	"context"
	"gobattle/common/api"
	"gobattle/framework/btconfig"
	"gobattle/framework/btlog"
	"gobattle/framework/btnet"
	"gobattle/framework/btregistry"
	"gobattle/framework/btrouter"
	"gobattle/framework/btrpc"
	"gobattle/game/config"
	"gobattle/game/module"
	"gobattle/game/repository"
	"gobattle/game/service"
	"gobattle/game/tpl"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/snowflake"
)

func main() {
	config.Load()
	cfg := config.Game

	tplMgr, err := tpl.Load(cfg.TplDir)
	if err != nil {
		panic(err)
	}
	db := openDB(cfg.DB)
	idNode, err := snowflake.NewNode(cfg.NodeId)
	if err != nil {
		panic(err)
	}

	queryTimeout := cfg.DB.QueryTimeout.Duration()
	rpcServer := btrpc.NewServer(cfg.Rpc, btrouter.JSONCodec{})
	dsRouter := btrouter.NewRouterHandler(btrouter.JSONCodec{})
	pushService := service.NewPushService(rpcServer, dsRouter)
	userService := service.NewUserService(db, idNode, queryTimeout)
	itemService := service.NewItemService(db, pushService, idNode, queryTimeout, tplMgr)
	shopService := service.NewShopService(db, tplMgr, itemService, idNode, queryTimeout)

	btrpc.RegisterBeforeBind[api.PeerApi](rpcServer, module.NewPeerApiModule(rpcServer), "Register")
	btrpc.Register[api.UserApi](rpcServer, module.NewUserApiModule(userService))
	btrpc.Register[api.ItemApi](rpcServer, module.NewItemApiModule(itemService))
	btrpc.Register[api.ShopApi](rpcServer, module.NewShopApiModule(shopService))
	btrpc.Register[api.GmApi](rpcServer, module.NewGmApiModule(itemService))
	if err := rpcServer.Start(); err != nil {
		panic(err)
	}

	var dsServer *btnet.ServerGroup
	if cfg.DsEnable {
		dsServer = newDsServer(cfg.DsServer, dsRouter, service.NewDsService())
		if err := dsServer.Server.Start(); err != nil {
			panic(err)
		}
	}

	registry, err := btregistry.NewZkRegistry(cfg.Zk)
	if err != nil {
		panic(err)
	}
	node := btrpc.Node{NodeId: cfg.NodeId, Addr: cfg.RpcAddr()}
	if err := registry.Register(cfg.Group, node); err != nil {
		panic(err)
	}
	btlog.Ins().Info("backendStarted", "nodeId", cfg.NodeId, "addr", node.Addr, "group", cfg.Group,
		"dsEnable", cfg.DsEnable, "queryTimeout", queryTimeout.String())

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()
	btlog.Ins().Info("backendStopping", "nodeId", cfg.NodeId)

	registry.Deregister()
	time.Sleep(time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	rpcServer.Shutdown(ctx)
	if dsServer != nil {
		dsServer.Shutdown(ctx)
	}
	registry.Close()
	if err := db.Close(); err != nil {
		btlog.Ins().Error("dbCloseError", "err", err)
	}
	btlog.Ins().Info("backendStopped", "nodeId", cfg.NodeId)
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

	boss := btnet.NewEventLoopGroup("ds-boss", 1)
	worker := btnet.NewEventLoopGroup("ds-worker", c.IOLoops())
	biz := btnet.NewEventLoopGroup("ds-biz", c.BizLoops())
	s := btnet.NewServer(c.Options(), boss, worker)
	connHandler := module.NewDsConnHandler(dsService)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, connHandler)
		p.AddLastGroup(biz, router)
	})
	return &btnet.ServerGroup{Name: "ds", Server: s, Groups: []*btnet.EventLoopGroup{boss, worker, biz}}
}
