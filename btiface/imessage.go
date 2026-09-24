package btiface

type IMessage interface {
	GetDataLen() uint32
	GetMsgId() uint32
	GetData() []byte
}
