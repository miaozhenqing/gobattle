package btrouter

import (
	"errors"
	"fmt"
	"gobattle/btlog"
	"gobattle/btnet"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"
)

var _ btnet.InHandler = (*Router)(nil)

type handler func(ctx *Context, req any) (resp any, err error)

type route struct {
	method string
	decode func(payload []byte) (any, error)
	call   handler // 业务方法
}

type Router struct {
	codec        Codec
	routes       map[uint32]*route
	interceptors []Interceptor
	frozen       atomic.Bool
}

func NewRouter(codec Codec) *Router {
	if codec == nil {
		codec = JSONCodec{}
	}
	return &Router{codec: codec, routes: make(map[uint32]*route)}
}

func (r *Router) AddInterceptor(its ...Interceptor) {
	if r.frozen.Load() {
		panic("use interceptor after dispatch started")
	}
	r.interceptors = append(r.interceptors, its...)
}

func Register[Req any, Resp any](r *Router, msgId uint32, fn func(*Context, *Req) (*Resp, error)) {
	if fn == nil {
		panic(fmt.Sprintf("handler of msgId %d is nil", msgId))
	}
	r.add(msgId, fn, decoder[Req](r), func(ctx *Context, req any) (any, error) {
		resp, err := fn(ctx, req.(*Req))
		if resp == nil {
			return nil, err
		}
		return resp, err
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

func (r *Router) add(msgId uint32, fn any, decode func([]byte) (any, error), call handler) {
	if r.frozen.Load() {
		panic(fmt.Sprintf("register msgId %d after dispatch started", msgId))
	}
	if msgId < MinRequestMsgId || msgId >= MinPushMsgId {
		panic(fmt.Sprintf("request msgId %d out of range [%d, %d)", msgId, MinRequestMsgId, MinPushMsgId))
	}
	if old, ok := r.routes[msgId]; ok {
		panic(fmt.Sprintf("msgId %d already registered by %s", msgId, old.method))
	}
	r.routes[msgId] = &route{method: funcName(fn), decode: decode, call: call}
}

func (r *Router) freeze() {
	if !r.frozen.Load() {
		r.frozen.Store(true)
	}
}

func funcName(fn any) string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	return strings.TrimSuffix(name, "-fm")
}

func (r *Router) OnRead(ctx *btnet.HandlerContext, msg any) {
	env, ok := msg.(*Envelope)
	if !ok {
		ctx.FireRead(msg)
		return
	}
	r.dispatch(ctx.GetConnection(), env)
}

func (r *Router) dispatch(conn *btnet.Connection, env *Envelope) {
	r.freeze()
	if env.Type != MsgTypeRequest {
		btlog.Ins().Warnf("unexpected msg type, connID:%v type:%v msgId:%v", conn.GetID(), env.Type, env.MsgId)
		return
	}

	rt := r.routes[env.MsgId]
	if rt == nil {
		btlog.Ins().Warnf("no route, connID:%v msgId:%v", conn.GetID(), env.MsgId)
		r.reply(conn, env, nil, errUnimplemented)
		return
	}

	ctx := &Context{Context: conn.Context(), conn: conn, msgId: env.MsgId, method: rt.method, startAt: time.Now()}
	resp, err := r.invoke(ctx, rt, env.Payload)
	r.reply(conn, env, resp, err)
}

func (r *Router) invoke(ctx *Context, rt *route, payload []byte) (resp any, err error) {
	defer r.recoverPanic(ctx, rt, &resp, &err)
	req, err := rt.decode(payload)
	if err != nil {
		return nil, err
	}

	n := 0
	for ; n < len(r.interceptors); n++ {
		if err = r.interceptors[n].Before(ctx, req); err != nil {
			break
		}
	}
	if err == nil {
		resp, err = r.call(ctx, rt, req)
	}
	for i := n - 1; i >= 0; i-- {
		r.interceptors[i].After(ctx, req, resp, err)
	}
	return resp, err
}

func (r *Router) call(ctx *Context, rt *route, req any) (resp any, err error) {
	defer r.recoverPanic(ctx, rt, &resp, &err)
	return rt.call(ctx, req)
}

func (r *Router) recoverPanic(ctx *Context, rt *route, resp *any, err *error) {
	if p := recover(); p != nil {
		btlog.Ins().Errorf("handler panic, connID:%v msgId:%v(%s) err:%v\n%s",
			ctx.conn.GetID(), ctx.msgId, rt.method, p, debug.Stack())
		*resp, *err = nil, errInternal(nil)
	}
}

func (r *Router) reply(conn *btnet.Connection, req *Envelope, resp any, err error) {
	e := r.toError(conn, req, err)
	if req.Seq == 0 { // 客户端不等结果，不回包

		return
	}

	out := &Envelope{Type: MsgTypeResponse, Seq: req.Seq, MsgId: req.MsgId}
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
	_ = conn.Write(out)
}

func (r *Router) toError(conn *btnet.Connection, env *Envelope, err error) *Error {
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
	case CodeUnimplemented: // dispatch 已打过 Warn
	default:
		btlog.Ins().Debugf("biz error, connID:%v msgId:%v err:%v", connID, msgId, e)
	}
	return e
}

func (r *Router) Push(conn *btnet.Connection, msgId uint32, v any) error {
	data, err := r.marshalPush(msgId, v)
	if err != nil {
		return err
	}
	return conn.Write(&Envelope{Type: MsgTypePush, MsgId: msgId, Payload: data})
}

func (r *Router) Broadcast(mgr *btnet.ConnManager, msgId uint32, v any) error {
	data, err := r.marshalPush(msgId, v)
	if err != nil {
		return err
	}
	// 整帧只编码一次，所有连接共享
	mgr.Broadcast(encodeEnvelope(&Envelope{Type: MsgTypePush, MsgId: msgId, Payload: data}))
	return nil
}

func (r *Router) marshalPush(msgId uint32, v any) ([]byte, error) {
	if msgId < MinPushMsgId {
		return nil, fmt.Errorf("push msgId %d < %d", msgId, MinPushMsgId)
	}
	return r.codec.Marshal(v)
}
