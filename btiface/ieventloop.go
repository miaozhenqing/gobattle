package btiface

type IEventLoop interface {
	Execute(task func())
}

type IEventLoopGroup interface {
	Next(key int64) IEventLoop
}
