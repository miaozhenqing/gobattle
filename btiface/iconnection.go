package btiface

import "context"

type IConnection interface {
	GetID() int64
	GetPipeline() IPipeline
	EventLoop() IEventLoop
	Stop()
	FlushAndStop()
	IsClosed() bool

	SetProperty(key, value any)
	GetProperty(key any) (any, bool)
	RemoveProperty(key any)

	Send(env *Envelope) error

	Context() context.Context
}
