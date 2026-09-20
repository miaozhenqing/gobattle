package btnet

import "gobattle/btiface"

type Request struct {
	conn btiface.IConnection
	data []byte
}

func NewRequest(conn btiface.IConnection, data []byte) *Request {
	return &Request{
		conn: conn,
		data: data,
	}
}
