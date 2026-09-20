package btiface

type IConnection interface {
	GetID() int64
	StartRead()
}
