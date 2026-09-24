package btiface

type IRequest interface {
	GetConnection() IConnection

	GetData() []byte

	HandleRouter(router IRouter)
}
