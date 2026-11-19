package btrpc

import (
	"cmp"
	"context"
	"gobattle/framework/btlog"
	"gobattle/framework/btrouter"
	"slices"
	"sync"
	"time"
)

type Invoker interface {
	Invoke(ctx context.Context, method string, payload []byte) ([]byte, error)
	Codec() btrouter.Codec
}

func Invoke[Resp any](ctx context.Context, inv Invoker, method string, req any) (*Resp, error) {
	payload, err := inv.Codec().Marshal(req)
	if err != nil {
		return nil, err
	}
	data, err := inv.Invoke(ctx, method, payload)
	if err != nil && len(data) == 0 {
		return nil, err
	}
	resp := new(Resp)
	if len(data) > 0 {
		if uErr := inv.Codec().Unmarshal(data, resp); uErr != nil {
			return nil, uErr
		}
	}
	return resp, err
}

type Discovery interface {
	Watch(group string, fn func(nodes []Node)) error
}

type ClusterConfig struct {
	Register      func(ctx context.Context, inv Invoker) error
	MaxPacketSize int
	CallTimeout   time.Duration
	Codec         btrouter.Codec
	OnPush        func(p *Packet)
}

type Cluster struct {
	cfg      ClusterConfig
	group    string
	balancer Balancer

	mu      sync.Mutex
	clients map[int64]*Client
	nodes   map[int64]Node
	closed  bool
}

func NewCluster(cfg ClusterConfig, d Discovery, group string, b Balancer) (*Cluster, error) {
	c := &Cluster{cfg: cfg, group: group, balancer: b,
		clients: map[int64]*Client{}, nodes: map[int64]Node{}}
	if c.cfg.Codec == nil {
		c.cfg.Codec = btrouter.JSONCodec{}
	}
	if err := d.Watch(group, c.syncDiscoveredNodes); err != nil {
		return nil, err
	}
	return c, nil
}

// 维护节点，下线的或者地址变动的或新增的
func (c *Cluster) syncDiscoveredNodes(nodes []Node) {
	latest := make(map[int64]Node, len(nodes))
	for _, node := range nodes {
		latest[node.NodeId] = node
	}
	var toClose []*Client
	var toConnect []Node
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	//需要关闭的旧链接
	for nodeId, oldNode := range c.nodes {
		newNode, exists := latest[nodeId]
		if !exists || oldNode.Addr != newNode.Addr {
			if client := c.clients[nodeId]; client != nil {
				delete(c.clients, nodeId)
				toClose = append(toClose, client)
			}
		}
	}
	//需要创建新链接
	for nodeId, node := range latest {
		old, exists := c.nodes[nodeId]
		if !exists || old.Addr != node.Addr {
			toConnect = append(toConnect, node)
		}
	}
	c.nodes = latest
	c.mu.Unlock()
	btlog.Ins().Info("rpcClusterNodesDiscovered", "group", c.group, "nodes", nodes, "toConnect", len(toConnect), "toClose", len(toClose))
	c.refreshBalancer()
	for _, client := range toClose {
		btlog.Ins().Info("rpcClusterNodeRemoved", "group", c.group, "nodeId", client.node.NodeId, "addr", client.node.Addr)
		client.Close()
	}
	for _, node := range toConnect {
		c.connectNode(node)
	}
}

func (c *Cluster) connectNode(n Node) {
	client, err := dialNodeClient(n, c.cfg.MaxPacketSize, c.cfg.Codec, c.cfg.OnPush)
	if err != nil {
		btlog.Ins().Error("rpcClusterDialFail", "group", c.group, "nodeId", n.NodeId, "addr", n.Addr, "err", err)
		return
	}
	if c.cfg.Register != nil {
		ctx, cancel := context.WithTimeout(context.Background(), c.cfg.CallTimeout)
		err = c.cfg.Register(ctx, client)
		cancel()
		if err != nil {
			btlog.Ins().Error("rpcClusterRegisterFail", "group", c.group, "nodeId", n.NodeId, "addr", n.Addr, "err", err)
			_ = client.Close()
			return
		}
	}
	c.mu.Lock()
	wanted, exists := c.nodes[n.NodeId]
	//确认下地址没变
	if c.closed || !exists || wanted.Addr != n.Addr || c.clients[n.NodeId] != nil {
		c.mu.Unlock()
		btlog.Ins().Warn("rpcClusterDiscardStale", "group", c.group, "nodeId", n.NodeId, "addr", n.Addr)
		_ = client.Close()
		return
	}
	c.clients[n.NodeId] = client
	c.mu.Unlock()
	btlog.Ins().Info("rpcClusterNodeConnected", "group", c.group, "nodeId", n.NodeId, "addr", n.Addr)
	c.refreshBalancer()
	go func() {
		<-client.Done()
		c.mu.Lock()
		if c.clients[n.NodeId] == client {
			delete(c.clients, n.NodeId)
		}
		c.mu.Unlock()
		btlog.Ins().Warn("rpcClusterNodeDisconnected", "group", c.group, "nodeId", n.NodeId, "addr", n.Addr)
		c.refreshBalancer()
	}()
}

func (c *Cluster) refreshBalancer() {
	c.mu.Lock()
	defer c.mu.Unlock()
	connected := make([]Node, 0, len(c.clients))
	for _, client := range c.clients {
		connected = append(connected, client.node)
	}
	slices.SortFunc(connected, func(a Node, b Node) int {
		return cmp.Compare(a.NodeId, b.NodeId)
	})
	c.balancer.Update(connected)
}

func (c *Cluster) getNodeClient(nodeId int64) (*Client, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	client, ok := c.clients[nodeId]
	return client, ok
}

func (c *Cluster) hasClosed() bool {

	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *Cluster) Codec() btrouter.Codec { return c.cfg.Codec }

func (c *Cluster) Invoke(ctx context.Context, method string, payload []byte) ([]byte, error) {
	if c.hasClosed() {
		btlog.Ins().Warn("rpcClusterClosed", "group", c.group, "rpcMethod", method)
		return nil, btrouter.CodeUnavailable
	}

	if _, has := ctx.Deadline(); !has {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.cfg.CallTimeout)
		defer cancel()
	}

	node, ok := c.balancer.Pick(ctx)
	if !ok {
		btlog.Ins().Warn("rpcClusterNoAvailableNode", "group", c.group, "rpcMethod", method, "userId", UserId(ctx))
		return nil, btrouter.CodeUnavailable
	}
	client, ok := c.getNodeClient(node.NodeId)
	if !ok || node.Addr != client.node.Addr {
		btlog.Ins().Warn("rpcClusterNodeClientMissing", "group", c.group, "rpcMethod", method, "nodeId", node.NodeId, "addr", node.Addr)
		return nil, btrouter.CodeUnavailable
	}
	return client.sendRequestAndWait(ctx, method, payload)
}

func (c *Cluster) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	clients := c.clients
	c.clients = map[int64]*Client{}
	c.nodes = map[int64]Node{}
	c.balancer.Update(nil)
	c.mu.Unlock()
	btlog.Ins().Info("rpcClusterClose", "group", c.group, "clients", len(clients))
	for _, cli := range clients {
		_ = cli.Close()
	}
}
