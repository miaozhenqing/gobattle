package btiface

type IConnManager interface {
	Add(conn IConnection)
	Remove(connId int64)
	Len() int
	CloseAll()
}
