package btnet

import (
	"gobattle/btiface"
	"sync"
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

var requestPool = sync.Pool{
	New: func() interface{} {
		return new(Request)
	},
}

func GetRequest(conn btiface.IConnection, data []byte) btiface.IRequest {
	req := requestPool.Get().(*Request)
	req.reset(conn, data)
	return req
}
func PutRequest(req btiface.IRequest) {
	if req == nil {
		return
	}
	request := req.(*Request)
	request.reset(nil, nil)
	requestPool.Put(request)
}

func (r *Request) reset(conn btiface.IConnection, data []byte) {
	r.conn = conn
	r.data = data
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
