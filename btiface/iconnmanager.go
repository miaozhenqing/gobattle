package btiface

type IConnManager interface {
	Add(conn iConnection)
	Remove(conn iConnection)
}
