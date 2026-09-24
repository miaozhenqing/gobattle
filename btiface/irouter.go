package btiface

type IRouter interface {
	Handle(req IRequest)
}
