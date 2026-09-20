package btnet

import "gobattle/btiface"

type ConnManager struct {
	connections map[int]btiface.IConnManager
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		connections: make(map[int]btiface.IConnManager),
	}
}

func (cm *ConnManager) Add(conn btiface.IConnection) {

}
func (cm *ConnManager) Remove(conn btiface.IConnection) {
}
