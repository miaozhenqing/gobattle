package btiface

type IPipeline interface {
	AddLast(h interface{}) IPipeline
	FireRead(msg interface{})
	FireWrite(msg interface{})

	AddLastGroup(g IEventLoopGroup, h interface{}) IPipeline
}

type PipelineInitializer func(p IPipeline)

type IHandler interface{}

type IInHandler interface {
	OnRead(ctx IHandlerContext, msg interface{})
}

type IOutHandler interface {
	OnWrite(ctx IHandlerContext, msg interface{})
}

type IHandlerContext interface {
	GetConnection() IConnection
	FireRead(msg interface{})
	FireWrite(msg interface{})
	FireActive()
	FireInactive()
	FireException(err error)
}

type IActiveHandler interface {
	OnActive(ctx IHandlerContext)
	OnInactive(ctx IHandlerContext)
}

type IExceptionHandler interface {
	OnException(ctx IHandlerContext, err error)
}
