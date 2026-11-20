package main

import (
	"context"
	"gobattle/common/api"
	"gobattle/framework/btlog"
	"gobattle/framework/btregistry"
	"gobattle/framework/btrouter"
	"gobattle/framework/btrpc"
	"gobattle/game/config"
	"gobattle/game/module"
	"gobattle/game/tpl"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/snowflake"
)

type app struct {
	cfg       *config.GameConfig
	tplMgr    *tpl.TplManager
	idNode    *snowflake.Node
	registry  *btregistry.ZkRegistry
	rpcServer *btrpc.Server
	clusters  *btrpc.ClusterManager
	stops     []func(ctx context.Context)
}

func newApp(cfg *config.GameConfig) *app {
	tplMgr, err := tpl.Load(cfg.TplDir)
	if err != nil {
		panic(err)
	}
	idNode, err := snowflake.NewNode(cfg.NodeId)
	if err != nil {
		panic(err)
	}
	registry, err := btregistry.NewZkRegistry(cfg.Zk)
	if err != nil {
		panic(err)
	}
	rpcServer := btrpc.NewServer(cfg.Rpc, btrouter.JSONCodec{})
	//目前所有模块都会被调用，所以都要放开注册
	btrpc.RegisterBeforeBind[api.PeerApi](rpcServer, module.NewPeerApiModule(rpcServer), "Register")

	return &app{cfg: cfg, tplMgr: tplMgr, idNode: idNode, registry: registry, rpcServer: rpcServer,
		clusters: newClusterManager(cfg, registry)}
}

func newClusterManager(cfg *config.GameConfig, registry *btregistry.ZkRegistry) *btrpc.ClusterManager {
	return btrpc.NewClusterManager(btrpc.ClusterConfig{

		Register: func(ctx context.Context, inv btrpc.Invoker) error {
			_, err := api.NewPeerApiClient(inv).Register(ctx, &api.RegisterPeerReq{PeerType: cfg.Role, PeerId: cfg.NodeId})
			return err
		},
		MaxPacketSize: cfg.RpcMaxPacketSize,
		CallTimeout:   cfg.CallTimeout.Duration(),
		Codec:         btrouter.JSONCodec{},
	}, registry)
}

func (a *app) onStop(fn func(ctx context.Context)) {
	a.stops = append(a.stops, fn)
}

func (a *app) run() {
	cfg := a.cfg
	if err := a.rpcServer.Start(); err != nil {
		panic(err)
	}
	node := btrpc.Node{NodeId: cfg.NodeId, Addr: cfg.RpcAddr()}
	// 先监听端口再注册 ZK：别人发现我的时候一定能连上
	if err := a.registry.Register(cfg.Role, node); err != nil {
		panic(err)
	}
	btlog.Ins().Info("backendStarted", "role", cfg.Role, "nodeId", cfg.NodeId, "addr", node.Addr,
		"dsEnable", cfg.DsEnable)

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-sigCtx.Done()
	btlog.Ins().Info("backendStopping", "role", cfg.Role, "nodeId", cfg.NodeId)

	// 先从zk移除节点，避免还有流量
	a.registry.Deregister()
	time.Sleep(time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	a.rpcServer.Shutdown(ctx)
	for i := len(a.stops) - 1; i >= 0; i-- {
		a.stops[i](ctx)
	}
	a.clusters.Close()
	a.registry.Close()
	btlog.Ins().Info("backendStopped", "role", cfg.Role, "nodeId", cfg.NodeId)
}
