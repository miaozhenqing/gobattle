package btiface

type IMessageHandler interface {
	HandleMessage(request IRequest)

	SetSerializer(s ISerializer)

	AddRouter(msgId uint32, r IRouter)

	AllocateWorkerID(connID int64) int
}
