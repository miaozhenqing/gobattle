package btiface

type IConnection interface {
	GetID() int64
	SendMsg(msgId uint32, data []byte) error
	GetWorkerID() int
	GetPipeline() IPipeline
	Stop()
	FlushAndStop()
	IsClosed() bool
}
