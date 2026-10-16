package btiface

type IDispatcher interface {
	Dispatch(req IRequest)
}
