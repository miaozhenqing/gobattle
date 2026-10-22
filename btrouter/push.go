package btrouter

import (
	"fmt"
	"gobattle/btiface"
)

func (r *Router) Push(conn btiface.IConnection, msgId uint32, v any) error {
	data, err := r.marshalPush(msgId, v)
	if err != nil {
		return err
	}
	return conn.Send(&btiface.Envelope{Type: btiface.MsgTypePush, MsgId: msgId, Payload: data})
}

func (r *Router) Broadcast(mgr btiface.IConnManager, msgId uint32, v any) error {
	data, err := r.marshalPush(msgId, v)
	if err != nil {
		return err
	}
	mgr.Broadcast(msgId, data)
	return nil
}

func (r *Router) marshalPush(msgId uint32, v any) ([]byte, error) {
	if msgId < MinPushMsgId {
		return nil, fmt.Errorf("push msgId %d < %d", msgId, MinPushMsgId)
	}
	return r.codec.Marshal(v)
}
