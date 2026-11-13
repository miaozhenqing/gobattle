package btrpc

import (
	"context"
	"sync/atomic"
)

type Node struct {
	NodeId int64  `json:"nodeId"`
	Addr   string `json:"addr"`
}

type Balancer interface {
	Update(nodes []Node)
	Pick(ctx context.Context) (Node, bool)
}

type roundRobin struct {
	nodes atomic.Pointer[[]Node]
	next  atomic.Uint64
}

func RoundRobin() Balancer { return &roundRobin{} }

func (b *roundRobin) Update(nodes []Node) {
	b.nodes.Store(&nodes)
}

func (b *roundRobin) Pick(ctx context.Context) (Node, bool) {
	nodes := b.nodes.Load()
	if nodes == nil || len(*nodes) == 0 {
		return Node{}, false
	}
	n := *nodes
	return n[b.next.Add(1)%uint64(len(n))], true
}
