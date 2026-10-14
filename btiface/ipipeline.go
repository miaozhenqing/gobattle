package btiface

type IPipeline interface {
	AddLast(name string, h interface{}) IPipeline
	FireRead(msg interface{})
	FireWrite(msg interface{})
}

type IHandler interface{}

type IInHandler interface {
	OnRead(ctx IHandlerContext, msg interface{})
}

type IOutHandler interface {
	OnWrite(ctx IHandlerContext, msg interface{})
}

type IHandlerContext interface {
	GetConnection() IConnection
	FireRead(msg interface{})
	FireWrite(msg interface{})
}
