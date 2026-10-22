package btiface

type IDispatcher interface {
	Dispatch(conn IConnection, env *Envelope)
}
