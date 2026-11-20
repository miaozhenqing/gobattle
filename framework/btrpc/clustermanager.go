package btrpc

import (
	"fmt"
	"gobattle/framework/btlog"
	"gobattle/framework/btrouter"
	"sync"
)

type ClusterManager struct {
	cfg       ClusterConfig
	discovery Discovery

	mu       sync.Mutex
	clusters map[string]*Cluster
	closed   bool
}

func NewClusterManager(cfg ClusterConfig, d Discovery) *ClusterManager {
	return &ClusterManager{cfg: cfg, discovery: d, clusters: make(map[string]*Cluster)}
}

func (m *ClusterManager) Get(group string, b Balancer) (*Cluster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, btrouter.CodeUnavailable
	}
	if c := m.clusters[group]; c != nil {
		return c, nil
	}
	c, err := NewCluster(m.cfg, m.discovery, group, b)
	if err != nil {
		return nil, err
	}
	m.clusters[group] = c
	btlog.Ins().Info("rpcClusterCreated", "group", group)
	return c, nil
}

func (m *ClusterManager) MustGet(group string, b Balancer) *Cluster {
	c, err := m.Get(group, b)
	if err != nil {
		panic(fmt.Sprintf("get cluster %s: %v", group, err))
	}
	return c
}

// 必须在 registry.Close 之前调用
func (m *ClusterManager) Close() {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return
	}
	m.closed = true
	clusters := m.clusters
	m.clusters = make(map[string]*Cluster)
	m.mu.Unlock()
	for _, c := range clusters {
		c.Close()
	}
}
