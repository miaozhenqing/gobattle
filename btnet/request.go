package btnet

import (
	"gobattle/btiface"
	"sync"
)

type Request struct {
	conn  btiface.IConnection
	msgId uint32
	data  []byte
}

func NewRequest(conn btiface.IConnection, msg btiface.IMessage) *Request {
	return &Request{
		conn:  conn,
		msgId: msg.GetMsgId(),
		data:  msg.GetData(),
	}
}

var requestPool = sync.Pool{
	New: func() interface{} {
		return new(Request)
	},
}

// GetRequest 池化获取，当前未使用；使用方需保证 req 不被异步持有，用完调用 PutRequest
func GetRequest(conn btiface.IConnection, msg btiface.IMessage) btiface.IRequest {
	req := requestPool.Get().(*Request)
	req.reset(conn, msg.GetMsgId(), msg.GetData())
	return req
}

func PutRequest(req btiface.IRequest) {
	if req == nil {
		return
	}
	request := req.(*Request)
	request.reset(nil, 0, nil)
	requestPool.Put(request)
}

func (r *Request) reset(conn btiface.IConnection, msgId uint32, data []byte) {
	r.conn = conn
	r.msgId = msgId
	r.data = data
}

func (r *Request) GetConnection() btiface.IConnection {
	return r.conn
}

func (r *Request) GetMsgId() uint32 {
	return r.msgId
}

func (r *Request) GetData() []byte {
	return r.data
}
