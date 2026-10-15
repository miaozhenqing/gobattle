package btiface

type IMessageHandler interface {
	HandleMessage(conn IConnection, frame []byte)

	AllocateWorkerID(connID int64) int
}
