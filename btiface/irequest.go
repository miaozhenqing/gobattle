package btiface

type IRequest interface {
	GetConnection() IConnection

	GetMsgId() uint32

	GetData() []byte
}
