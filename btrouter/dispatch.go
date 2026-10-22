package btrouter

import (
	"errors"
	"gobattle/btiface"
	"gobattle/btlog"
	"runtime/debug"
)

func (r *Router) Dispatch(conn btiface.IConnection, env *btiface.Envelope) {
	r.frozen.Store(true)
	if env.Type != btiface.MsgTypeRequest {
		btlog.Ins().Warnf("unexpected msg type, connID:%v type:%v msgId:%v", conn.GetID(), env.Type, env.MsgId)
		return
	}

	rt := r.routes[env.MsgId]
	if rt == nil {
		btlog.Ins().Warnf("no route, connID:%v msgId:%v", conn.GetID(), env.MsgId)
		r.reply(conn, env, nil, errUnimplemented, false)
		return
	}

	ctx := &Context{Context: conn.Context(), conn: conn, msgId: env.MsgId, method: rt.method}
	resp, err := r.invoke(ctx, rt, env.Payload)
	r.reply(conn, env, resp, err, rt.oneway)
}

func (r *Router) invoke(ctx *Context, rt *route, payload []byte) (resp any, err error) {
	defer func() {
		if p := recover(); p != nil {
			btlog.Ins().Errorf("handler panic, connID:%v msgId:%v(%s) err:%v\n%s",
				ctx.conn.GetID(), ctx.msgId, rt.method, p, debug.Stack())
			resp, err = nil, errInternal(nil)
		}
	}()
	return rt.h(ctx, payload)
}

func (r *Router) reply(conn btiface.IConnection, req *btiface.Envelope, resp any, err error, oneway bool) {
	e := r.toError(conn, req, err)
	if oneway || req.Seq == 0 {
		return
	}

	out := &btiface.Envelope{Type: btiface.MsgTypeResponse, Seq: req.Seq, MsgId: req.MsgId}
	if e != nil {
		out.Code, out.Payload = e.Code, []byte(truncate(e.Msg, maxErrMsgLen))
	} else if resp != nil {
		b, mErr := r.codec.Marshal(resp)
		if mErr != nil {
			btlog.Ins().Errorf("marshal resp error, connID:%v msgId:%v err:%v", conn.GetID(), req.MsgId, mErr)
			ie := errInternal(nil)
			out.Code, out.Payload = ie.Code, []byte(ie.Msg)
		} else {
			out.Payload = b
		}
	}
	_ = conn.Send(out)
}

func (r *Router) toError(conn btiface.IConnection, env *btiface.Envelope, err error) *Error {
	if err == nil {
		return nil
	}
	connID, msgId := conn.GetID(), env.MsgId
	var e *Error
	if !errors.As(err, &e) {
		btlog.Ins().Errorf("handler error, connID:%v msgId:%v err:%v", connID, msgId, err)
		return errInternal(err)
	}
	switch e.Code {
	case CodeInternal:
		if e.cause != nil { // panic 在 invoke 里已打过，cause 为 nil
			btlog.Ins().Errorf("internal error, connID:%v msgId:%v err:%v", connID, msgId, e)
		}
	case CodeInvalidArgument:
		btlog.Ins().Warnf("invalid argument, connID:%v msgId:%v err:%v", connID, msgId, e)
	case CodeUnimplemented: // Dispatch 已打过 Warn
	default:
		btlog.Ins().Debugf("biz error, connID:%v msgId:%v err:%v", connID, msgId, e)
	}
	return e
}
