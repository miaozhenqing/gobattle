package btiface

import "iter"

type IConnManager interface {
	Get(connID int64) (IConnection, bool)
	Len() int
	All() iter.Seq[IConnection]
	Broadcast(msgId uint32, data []byte)
}
