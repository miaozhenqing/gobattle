package btnet

import (
	"gobattle/btiface"
	"sync"
)

type ConnManager struct {
	connections map[int64]btiface.IConnection
	mu          sync.RWMutex
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		connections: make(map[int64]btiface.IConnection),
	}
}

func (cm *ConnManager) Add(conn btiface.IConnection) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.connections[conn.GetID()] = conn
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

func (cm *ConnManager) CloseAll() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	for _, connection := range cm.connections {
		connection.Stop()
	}
}
