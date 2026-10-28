package model

import (
	"gobattle/btnet"
	"sync/atomic"
)

const (
	DsStateIdle int32 = 0
	DsStateBusy int32 = 1
)

type DsModel struct {
	ID           int64
	IP           string
	Port         int
	RegisterTime int64
	Conn         *btnet.Connection
	State        atomic.Int32
}
