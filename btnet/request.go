package btnet

import (
	"gobattle/btiface"
)

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

func (r *Request) GetConnection() btiface.IConnection {
	return r.conn
}
func (r *Request) GetData() []byte {
	return r.data
}

func (r *Request) HandleRouter(router btiface.IRouter) {
	router.Handle(r)
}
