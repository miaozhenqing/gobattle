package btiface

// Deprecated: 由 IEventLoopGroup 取代
type IMessageHandler interface {
	HandleMessage(conn IConnection, msg IMessage)

	AllocateWorkerID(connID int64) int
}
