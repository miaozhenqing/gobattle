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
	"gobattle/framework/btsession"
	"gobattle/gateway/config"
	"gobattle/gateway/module"
	"gobattle/gateway/service"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	config.Load()
	cfg := config.Gateway

	sessionMgr := btsession.NewManager()
	router := btrouter.NewRouterHandler(btrouter.JSONCodec{})
	router.AddInterceptor(
		btrouter.RateLimit(20, 40),
		btrouter.Auth(btsession.IsLoggedIn),
		btrouter.SlowLog(50*time.Millisecond),
	)
	pushService := service.NewPushService(router, sessionMgr)

	registry, err := btregistry.NewZkRegistry(cfg.Zk)
	if err != nil {
		panic(err)
	}
	//创建logic的集群信息
	logicCluster, err := btrpc.NewCluster(btrpc.ClusterConfig{
		Register: func(ctx context.Context, inv btrpc.Invoker) error {
			_, err := api.NewPeerApiClient(inv).Register(ctx, &api.RegisterPeerReq{PeerType: "gateway", PeerId: cfg.NodeId})
			return err
		},
		MaxPacketSize: cfg.RpcMaxPacketSize,
		CallTimeout:   cfg.CallTimeout.Duration(),
		Codec:         btrouter.JSONCodec{},
		OnPush:        pushService.OnBackendPush,
	}, registry, "logic", btrpc.RoundRobin())
	if err != nil {
		btlog.Ins().Error("newClusterError", "err", err)
		panic(err)
	}
	//
	router.RegisterModules(
		module.NewLoginModule(api.NewUserApiClient(logicCluster), pushService, sessionMgr),
		module.NewItemModule(api.NewItemApiClient(logicCluster)),
		module.NewShopModule(api.NewShopApiClient(logicCluster)),
		module.NewGmModule(api.NewGmApiClient(logicCluster)),
	)
	gw := newGatewayServer(cfg.Server, router, sessionMgr)
	if err := gw.Server.Start(); err != nil {
		panic(err)
	}
	btlog.Ins().Info("gatewayStarted", "nodeId", cfg.NodeId, "port", cfg.Server.TcpPort,
		"callTimeout", cfg.CallTimeout.Duration().String(), "rpcMaxPacketSize", cfg.RpcMaxPacketSize)

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()
	btlog.Ins().Info("gatewayStopping", "nodeId", cfg.NodeId)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	gw.Shutdown(ctx)
	logicCluster.Close()
	registry.Close()
	btlog.Ins().Info("gatewayStopped", "nodeId", cfg.NodeId)
}

func newGatewayServer(c btconfig.ServerConfig, router *btrouter.RouterHandler, sessionMgr *btsession.Manager) *btnet.ServerGroup {
	boss := btnet.NewEventLoopGroup("gw-boss", 1)
	worker := btnet.NewEventLoopGroup("gw-worker", c.IOLoops())
	biz := btnet.NewEventLoopGroup("gw-biz", c.BizLoops())
	s := btnet.NewServer(c.Options(), boss, worker)
	sessionHandler := btsession.NewSessionHandler(sessionMgr)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, sessionHandler)
		p.AddLastGroup(biz, router)
	})
	return &btnet.ServerGroup{Name: "gateway", Server: s, Groups: []*btnet.EventLoopGroup{boss, worker, biz}}
}
