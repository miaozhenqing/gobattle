package main

import (
	"context"
	"gobattle/common/api"
	commonredis "gobattle/common/redis"
	"gobattle/common/redis/sessionstore"
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
	redisClient := commonredis.NewClient(cfg.Redis)
	defer redisClient.Close()
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 2*time.Second)
	err := redisClient.Ping(pingCtx).Err()
	pingCancel()
	if err != nil {
		panic(err)
	}

	sessionMgr := btsession.NewManager()
	router := btrouter.NewRouterHandler(btrouter.JSONCodec{})
	pushService := service.NewPushService(router, sessionMgr)
	kickService := service.NewKickService(pushService)
	store := sessionstore.NewStore(redisClient, cfg.SessionTTL.Duration())
	onlineService := service.NewOnlineService(redisClient, store, sessionMgr, kickService, pushService,
		cfg.NodeId, cfg.SessionTTL.Duration(), cfg.ReconcileInterval.Duration())
	sessionMgr.OnUnbind(onlineService.OnUnbind)
	router.AddInterceptor(
		btrouter.RateLimit(20, 40),
		btrouter.Auth(onlineService.IsActiveSession),
		btrouter.SlowLog(50*time.Millisecond),
	)

	registry, err := btregistry.NewZkRegistry(cfg.Zk)
	if err != nil {
		panic(err)
	}
	clusters := btrpc.NewClusterManager(btrpc.ClusterConfig{
		Register: func(ctx context.Context, inv btrpc.Invoker) error {
			_, err := api.NewPeerApiClient(inv).Register(ctx, &api.RegisterPeerReq{PeerType: "gateway", PeerId: cfg.NodeId})
			return err
		},
		MaxPacketSize: cfg.RpcMaxPacketSize,
		CallTimeout:   cfg.CallTimeout.Duration(),
		Codec:         btrouter.JSONCodec{},
		OnPush:        pushService.OnBackendPush,
	}, registry)

	logicCluster := clusters.MustGet(api.GroupLogic, btrpc.RoundRobin())

	router.RegisterModules(
		module.NewLoginModule(api.NewUserApiClient(logicCluster), sessionMgr, onlineService, kickService),
		module.NewItemModule(api.NewItemApiClient(logicCluster)),
		module.NewShopModule(api.NewShopApiClient(logicCluster)),
		module.NewGmModule(api.NewGmApiClient(logicCluster)),
	)
	if err := onlineService.Start(); err != nil {
		clusters.Close()
		registry.Close()
		panic(err)
	}
	gw := newGatewayServer(cfg.Server, router, sessionMgr)
	if err := gw.Server.Start(); err != nil {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		onlineService.Stop(stopCtx)
		cancel()
		clusters.Close()
		registry.Close()
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
	onlineService.Stop(ctx)
	clusters.Close()
	registry.Close()
	btlog.Ins().Info("gatewayStopped", "nodeId", cfg.NodeId)
}

func newGatewayServer(c btconfig.ServerConfig, router *btrouter.RouterHandler, sessionMgr *btsession.Manager) *btnet.ServerGroup {
	worker := btnet.NewEventLoopGroup("gw-worker", c.IOLoops())
	biz := btnet.NewEventLoopGroup("gw-biz", c.BizLoops())
	s := btnet.NewServer(c.Options(), worker)
	sessionHandler := btsession.NewSessionHandler(sessionMgr)
	s.SetPipelineInitializer(func(p *btnet.Pipeline) {
		p.AddLast(btrouter.NewEnvelopeCodec())
		p.AddLastGroup(biz, sessionHandler)
		p.AddLastGroup(biz, router)
	})
	return &btnet.ServerGroup{Name: "gateway", Server: s, Groups: []*btnet.EventLoopGroup{worker, biz}}
}
