package btiface

type IConnection interface {
	GetID() int
	StartRead()
}
