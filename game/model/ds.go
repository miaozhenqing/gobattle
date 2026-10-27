package model

import "gobattle/btnet"

type DsModel struct {
	ID           int64
	IP           string
	Port         int
	RegisterTime int64
	Conn         *btnet.Connection
}
