package btregistry

import (
	"encoding/json"
	"errors"
	"gobattle/framework/btconfig"
	"gobattle/framework/btlog"
	"gobattle/framework/btrpc"
	"maps"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

var _ btrpc.Discovery = (*ZkRegistry)(nil)

type ZkRegistry struct {
	conn *zk.Conn
	root string
	mu   sync.Mutex
	//临时路径 -> node data；会话过期后补注册；支持同个进程多个实例
	nodeData map[string][]byte
	closed   chan struct{}
	once     sync.Once
}

func NewZkRegistry(cfg btconfig.ZkConfig) (*ZkRegistry, error) {
	connect, events, err := zk.Connect(cfg.Addrs, cfg.SessionTimeout.Duration())
	if err != nil {
		return nil, err
	}
	zr := &ZkRegistry{
		conn:     connect,
		root:     cfg.Root,
		nodeData: make(map[string][]byte),
		closed:   make(chan struct{}),
	}
	btlog.Ins().Info("zkConnect", "addrs", cfg.Addrs, "root", cfg.Root, "sessionTimeout", cfg.SessionTimeout.Duration().String())
	go zr.watchSessionEvents(events)
	return zr, nil
}

func (r *ZkRegistry) backendGroupPath(group string) string {
	return path.Join(r.root, "backend", group)
}

// 创建 /{root}/backend/{group}/{nodeId} 临时节点
func (r *ZkRegistry) Register(group string, node btrpc.Node) error {
	groupPath := r.backendGroupPath(group)
	if err := r.ensurePersistentPath(groupPath); err != nil {
		return err
	}
	p := path.Join(groupPath, strconv.FormatInt(node.NodeId, 10))
	data, _ := json.Marshal(node)
	err := r.createEphemeralNode(p, data)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.nodeData[p] = data
	r.mu.Unlock()
	btlog.Ins().Info("zkRegistered", "path", p, "addr", node.Addr)
	return nil
}

func (r *ZkRegistry) createEphemeralNode(p string, data []byte) error {
	_, err := r.conn.Create(p, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	//可能有上次还未过期的节点，这里删了重建
	if errors.Is(err, zk.ErrNodeExists) {
		btlog.Ins().Warn("zkNodeExistsRecreate", "path", p)
		r.conn.Delete(p, -1)
		_, err = r.conn.Create(p, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	}
	return err
}

// 逐级创建持久节点
func (r *ZkRegistry) ensurePersistentPath(p string) error {
	p = strings.TrimPrefix(p, "/")
	cur := ""
	for _, part := range strings.Split(p, "/") {
		cur = cur + "/" + part
		_, err := r.conn.Create(cur, nil, zk.FlagPersistent, zk.WorldACL(zk.PermAll))
		if err != nil && !errors.Is(err, zk.ErrNodeExists) {
			return err
		}

	}
	return nil
}

// 删除全部节点
func (r *ZkRegistry) Deregister() {
	r.mu.Lock()
	collect := slices.Collect(maps.Keys(r.nodeData))
	r.nodeData = make(map[string][]byte)
	r.mu.Unlock()
	for _, p := range collect {
		err := r.conn.Delete(p, -1)
		if err != nil && !errors.Is(err, zk.ErrNoNode) {
			btlog.Ins().Error("zkDeregisterError", "path", p, "err", err)
			continue
		}
		btlog.Ins().Info("zkDeregistered", "path", p)
	}
}

// 会话过期重连后补注册
func (r *ZkRegistry) watchSessionEvents(events <-chan zk.Event) {
	expired := false
	for {
		select {
		case <-r.closed:
			return
		case e, ok := <-events:
			if !ok {
				return
			}
			if e.Type == zk.EventSession {
				btlog.Ins().Info("zkSessionEvent", "state", e.State.String(), "server", e.Server, "sessionId", r.conn.SessionID())
			}
			switch e.State {
			case zk.StateExpired:
				expired = true
				btlog.Ins().Warn("zkSessionExpired")
			case zk.StateHasSession:
				if expired {
					expired = false
					r.restoreRegisteredNodes()
				}
			}
		}
	}
}

func (r *ZkRegistry) restoreRegisteredNodes() {
	r.mu.Lock()
	owned := maps.Clone(r.nodeData)
	r.mu.Unlock()
	for p, data := range owned {
		if err := r.createEphemeralNode(p, data); err != nil {
			btlog.Ins().Error("zkReRegisterError", "path", p, "err", err)
			continue
		}
		btlog.Ins().Info("zkReRegistered", "path", p)
	}
}

// 监听子节点
func (r *ZkRegistry) Watch(group string, fn func([]btrpc.Node)) error {
	dir := r.backendGroupPath(group)
	if err := r.ensurePersistentPath(dir); err != nil {
		return err
	}
	children, _, events, err := r.conn.ChildrenW(dir)
	if err != nil {
		return err
	}
	btlog.Ins().Info("zkWatch", "group", group, "path", dir, "children", children)
	//首次直接返回子节点信息
	fn(r.readGroupNodes(dir, children))
	//再启动一个监听，直到受到关闭信号
	go func() {
		for {
			select {
			case <-events:

			case <-r.closed:
				//收到 Registry 关闭信号，退出监听
				return
			}
			for {
				//golang zk监听是一次性的，需要重新注册
				children, _, events, err = r.conn.ChildrenW(dir)
				if err == nil {
					break
				}
				btlog.Ins().Error("zkWatchError", "group", group, "err", err)
				select {
				case <-time.After(time.Second):
				case <-r.closed:
					return
				}
			}
			btlog.Ins().Info("zkChildrenChanged", "group", group, "path", dir, "children", children)
			fn(r.readGroupNodes(dir, children))
		}
	}()
	return nil
}

func (r *ZkRegistry) readGroupNodes(dir string, children []string) []btrpc.Node {
	nodes := make([]btrpc.Node, 0, len(children))
	for _, name := range children {
		data, _, err := r.conn.Get(path.Join(dir, name))
		if err != nil {
			btlog.Ins().Warn("zkGetNodeError", "path", path.Join(dir, name), "err", err)
			continue
		}
		var n btrpc.Node
		err = json.Unmarshal(data, &n)
		if err != nil {
			btlog.Ins().Error("zkNodeDataError", "path", path.Join(dir, name), "err", err)
			continue
		}
		nodes = append(nodes, n)
	}
	return nodes
}

func (r *ZkRegistry) Close() {
	r.once.Do(func() {
		btlog.Ins().Info("zkClose")
		close(r.closed)
		r.conn.Close()
	})
}
