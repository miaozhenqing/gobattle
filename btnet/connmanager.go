package btnet

import (
	"gobattle/btiface"
	"iter"
	"maps"
	"slices"
	"sync"
)

type ConnManager struct {
	connections map[int64]btiface.IConnection
	mu          sync.RWMutex
	closed      bool
	wg          sync.WaitGroup
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		connections: make(map[int64]btiface.IConnection),
	}
}

func (cm *ConnManager) Add(conn btiface.IConnection) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.closed {
		return false
	}
	cm.wg.Add(1)
	cm.connections[conn.GetID()] = conn
	return true
}

func (cm *ConnManager) done() {
	cm.wg.Done()
}

func (cm *ConnManager) wait() {
	cm.wg.Wait()
}

func (cm *ConnManager) Get(connId int64) (btiface.IConnection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	connection, ok := cm.connections[connId]
	return connection, ok
}

func (cm *ConnManager) All() iter.Seq[btiface.IConnection] {
	return func(yield func(btiface.IConnection) bool) {
		for _, c := range cm.snapshot() {
			if !yield(c) {
				return
			}
		}
	}
}

func (cm *ConnManager) Broadcast(msgId uint32, data []byte) {
	// 信封只读，所有连接共享同一个
	env := &btiface.Envelope{Type: btiface.MsgTypePush, MsgId: msgId, Payload: data}
	for c := range cm.All() {
		_ = c.Send(env) // 已关闭返回 ErrConnClosed，忽略
	}
}

func (cm *ConnManager) snapshot() []btiface.IConnection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return slices.Collect(maps.Values(cm.connections))
}

func (cm *ConnManager) Remove(connId int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.connections, connId)
}
func (cm *ConnManager) Len() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.connections)
}

func (cm *ConnManager) close() []btiface.IConnection {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.closed = true
	return slices.Collect(maps.Values(cm.connections))
}

func (cm *ConnManager) CloseAll() {
	for _, c := range cm.close() {
		c.Stop()
	}
}
