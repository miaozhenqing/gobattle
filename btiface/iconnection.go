package btiface

type IConnection interface {
	GetID() int64
	SendMsg(msgId uint32, data []byte)
	GetWorkerID() int
	GetPipeline() IPipeline
	Stop()
}
