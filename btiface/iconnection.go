package btiface

type IConnection interface {
	GetID() int64
	SendMsg(msgId int32, msg []byte)
}
