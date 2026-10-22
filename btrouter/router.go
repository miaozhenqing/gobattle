package btrouter

import (
	"errors"
	"fmt"
	"gobattle/btiface"
	"gobattle/btlog"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
)

var _ btiface.IDispatcher = (*Router)(nil)

type Handler func(ctx *Context, req any) (resp any, err error)

type Middleware func(ctx *Context, req any, next Handler) (resp any, err error)

type route struct {
	method string
	decode func(payload []byte) (any, error)
	call   Handler // 业务方法
	h      Handler // 冻结时组装
	oneway bool
}

type Router struct {
	codec  Codec
	routes map[uint32]*route
	mws    []Middleware
	frozen atomic.Bool
	once   sync.Once
}

func NewRouter(codec Codec) *Router {
	if codec == nil {
		codec = JSONCodec{}
	}
	return &Router{codec: codec, routes: make(map[uint32]*route)}
}

func Register[Req, Resp any](r *Router, msgId uint32, fn func(*Context, *Req) (*Resp, error)) {
	if fn == nil {
		panic(fmt.Sprintf("handler of msgId %d is nil", msgId))
	}
	r.add(msgId, fn, false, decoder[Req](r), func(ctx *Context, req any) (any, error) {
		resp, err := fn(ctx, req.(*Req))
		if resp == nil {
			return nil, err
		}
		return resp, err
	})
}

func RegisterOneway[Req any](r *Router, msgId uint32, fn func(*Context, *Req) error) {
	if fn == nil {
		panic(fmt.Sprintf("handler of msgId %d is nil", msgId))
	}
	r.add(msgId, fn, true, decoder[Req](r), func(ctx *Context, req any) (any, error) {
		return nil, fn(ctx, req.(*Req))
	})
}

func decoder[Req any](r *Router) func([]byte) (any, error) {
	return func(payload []byte) (any, error) {
		req := new(Req)
		if err := r.unmarshal(payload, req); err != nil {
			return nil, err
		}
		return req, nil
	}
}

func (r *Router) unmarshal(payload []byte, req any) error {
	if len(payload) == 0 { // 无参请求可不带 payload
		return nil
	}
	if err := r.codec.Unmarshal(payload, req); err != nil {
		return errInvalidArgument(err)
	}
	return nil
}

func (r *Router) add(msgId uint32, fn any, oneway bool, decode func([]byte) (any, error), call Handler) {
	if r.frozen.Load() {
		panic(fmt.Sprintf("register msgId %d after dispatch started", msgId))
	}
	if msgId < MinRequestMsgId || msgId >= MinPushMsgId {
		panic(fmt.Sprintf("request msgId %d out of range [%d, %d)", msgId, MinRequestMsgId, MinPushMsgId))
	}
	if old, ok := r.routes[msgId]; ok {
		panic(fmt.Sprintf("msgId %d already registered by %s", msgId, old.method))
	}
	r.routes[msgId] = &route{method: funcName(fn), decode: decode, call: call, oneway: oneway}
}

func (r *Router) Use(mws ...Middleware) {
	if r.frozen.Load() {
		panic("use middleware after dispatch started")
	}
	for _, mw := range mws {
		if mw == nil {
			panic("middleware is nil")
		}
	}
	r.mws = append(r.mws, mws...)
}

func (r *Router) freeze() {
	r.once.Do(func() {
		r.frozen.Store(true)
		for _, rt := range r.routes {
			rt.h = chain(r.mws, rt.call)
		}
	})
}

func chain(mws []Middleware, h Handler) Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		mw, next := mws[i], h
		h = func(ctx *Context, req any) (any, error) {
			return mw(ctx, req, next)
		}
	}
	return h
}

func funcName(fn any) string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	return strings.TrimSuffix(name, "-fm")
}

func (r *Router) Dispatch(conn btiface.IConnection, env *btiface.Envelope) {
	r.freeze()
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
	req, err := rt.decode(payload)
	if err != nil {
		return nil, err
	}
	return rt.h(ctx, req)
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
