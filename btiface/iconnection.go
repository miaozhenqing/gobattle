package btiface

type IConnection interface {
	GetID() int64
	SendMsg(msgId uint32, data []byte) error
	GetPipeline() IPipeline
	EventLoop() IEventLoop
	Stop()
	FlushAndStop()
	IsClosed() bool

	SetProperty(key, value any)
	GetProperty(key any) (any, bool)
	RemoveProperty(key any)
}
