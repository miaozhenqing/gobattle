package btiface

type IConnManager interface {
	Add(conn IConnection)
	Remove(conn IConnection)
}
