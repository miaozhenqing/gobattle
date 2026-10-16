package btiface

type IMessageHandler interface {
	HandleMessage(conn IConnection, msg IMessage)

	AllocateWorkerID(connID int64) int
}
