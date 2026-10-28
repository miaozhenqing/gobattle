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

type route struct {
	method string
	public bool // 免登录
	decode func(payload []byte) (any, error)
	call   func(ctx *RequestContext, req any) (any, error) // 业务方法
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
	for _, it := range its {
		if it == nil {
			panic("interceptor is nil")
		}
	}
	r.interceptors = append(r.interceptors, its...)
}

type Module interface {
	RegisterRoutes(r *Router)
}

func (r *Router) RegisterModules(ms ...Module) {
	for _, m := range ms {
		m.RegisterRoutes(r)
	}
}

func Handle[Req any, Resp any](r *Router, msgId uint32, fn func(*RequestContext, *Req) (*Resp, error)) {
	r.addRoute(msgId, newRoute(r, msgId, fn, false))
}

// HandlePublic 注册免登录路由（登录、心跳）
func HandlePublic[Req any, Resp any](r *Router, msgId uint32, fn func(*RequestContext, *Req) (*Resp, error)) {
	r.addRoute(msgId, newRoute(r, msgId, fn, true))
}

func newRoute[Req any, Resp any](r *Router, msgId uint32, fn func(*RequestContext, *Req) (*Resp, error), public bool) *route {
	if fn == nil {
		panic(fmt.Sprintf("handler of msgId %d is nil", msgId))
	}
	return &route{
		method: funcName(fn),
		public: public,
		decode: func(payload []byte) (any, error) {
			req := new(Req)
			if err := r.unmarshal(payload, req); err != nil {
				return nil, err
			}
			return req, nil
		},
		call: func(ctx *RequestContext, req any) (any, error) {
			resp, err := fn(ctx, req.(*Req))
			if resp == nil {
				return nil, err
			}
			return resp, err
		},
	}
}

func (r *Router) unmarshal(payload []byte, req any) error {
	if len(payload) == 0 { // 无参请求可不带 payload
		return nil
	}
	return r.codec.Unmarshal(payload, req)
}

func (r *Router) addRoute(msgId uint32, rt *route) {
	if r.frozen.Load() {
		panic(fmt.Sprintf("register msgId %d after dispatch started", msgId))
	}
	if old, ok := r.routes[msgId]; ok {
		panic(fmt.Sprintf("msgId %d already registered by %s", msgId, old.method))
	}
	r.routes[msgId] = rt
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
		r.reply(conn, env, nil, CodeUnimplemented)
		return
	}

	ctx := &RequestContext{Context: conn.Context(), conn: conn, msgId: env.MsgId, route: rt, startAt: time.Now()}
	resp, err := r.invoke(ctx, rt, env.Payload)
	r.reply(conn, env, resp, err)
}

func (r *Router) invoke(ctx *RequestContext, rt *route, payload []byte) (resp any, err error) {
	req, err := r.safeDecode(ctx, rt, payload)
	if err != nil {
		return nil, err
	}

	n := 0
	for ; n < len(r.interceptors); n++ {
		if err = r.safeBefore(ctx, rt, r.interceptors[n], req); err != nil {
			break
		}
	}
	if err == nil {
		resp, err = r.safeCall(ctx, rt, req)
		btlog.Ins().Infof(">>>>>>>>>>invoke msgId:%v, method:%v", ctx.msgId, rt.method)
	}
	for i := n - 1; i >= 0; i-- {
		r.safeAfter(ctx, rt, r.interceptors[i], req, resp, err)
	}
	return resp, err
}

func (r *Router) safeDecode(ctx *RequestContext, rt *route, payload []byte) (req any, err error) {
	defer r.recoverPanic(ctx, rt, "decode", nil, &err)
	if req, err = rt.decode(payload); err != nil {
		btlog.Ins().Warnf("decode error, connID:%v msgId:%v(%s) err:%v", ctx.conn.GetID(), ctx.msgId, rt.method, err)
		return nil, CodeInvalidArgument
	}
	return req, nil
}

func (r *Router) safeBefore(ctx *RequestContext, rt *route, it Interceptor, req any) (err error) {
	defer r.recoverPanic(ctx, rt, "interceptor.Before", it, &err)
	return it.Before(ctx, req)
}

func (r *Router) safeCall(ctx *RequestContext, rt *route, req any) (resp any, err error) {
	defer r.recoverPanic(ctx, rt, "handler", nil, &err)
	return rt.call(ctx, req)
}

func (r *Router) safeAfter(ctx *RequestContext, rt *route, it Interceptor, req, resp any, err error) {
	var ignored error
	defer r.recoverPanic(ctx, rt, "interceptor.After", it, &ignored)
	it.After(ctx, req, resp, err)
}

func (r *Router) recoverPanic(ctx *RequestContext, rt *route, stage string, it Interceptor, err *error) {
	if p := recover(); p != nil {
		if it != nil {
			stage = fmt.Sprintf("%s(%T)", stage, it)
		}
		btlog.Ins().Errorf("%s panic, connID:%v msgId:%v(%s) err:%v\n%s",
			stage, ctx.conn.GetID(), ctx.msgId, rt.method, p, debug.Stack())
		*err = CodeInternal
	}
}

func (r *Router) reply(conn *btnet.Connection, req *Envelope, resp any, err error) {
	code := r.toCode(conn, req, err)
	if req.Seq == 0 {
		return
	}

	out := &Envelope{Type: MsgTypeResponse, Seq: req.Seq, MsgId: req.MsgId, Code: uint32(code)}
	if code == CodeOK && resp != nil {
		b, mErr := r.codec.Marshal(resp)
		if mErr != nil {
			btlog.Ins().Errorf("marshal resp error, connID:%v msgId:%v err:%v", conn.GetID(), req.MsgId, mErr)
			out.Code = uint32(CodeInternal)
		} else {
			out.Payload = b
		}
	}
	_ = conn.Write(out)
}

func (r *Router) toCode(conn *btnet.Connection, env *Envelope, err error) Code {
	if err == nil {
		return CodeOK
	}
	var code Code
	if errors.As(err, &code) {
		// 包装过的 Code 带有上下文（如底层 DB 错误），日志打完整链，客户端只拿到 code
		if err != error(code) {
			btlog.Ins().Warnf("code error, connID:%v msgId:%v code:%v err:%v", conn.GetID(), env.MsgId, uint32(code), err)
		}
		return code
	}
	btlog.Ins().Errorf("non-code error, connID:%v msgId:%v err:%v", conn.GetID(), env.MsgId, err)
	return CodeInternal
}

func (r *Router) Push(conn *btnet.Connection, msgId uint32, v any) error {
	data, err := r.codec.Marshal(v)
	if err != nil {
		return err
	}
	return conn.Write(&Envelope{Type: MsgTypePush, MsgId: msgId, Payload: data})
}

func (r *Router) Broadcast(mgr *btnet.ConnManager, msgId uint32, v any) error {
	data, err := r.codec.Marshal(v)
	if err != nil {
		return err
	}
	// 编码一次
	mgr.Broadcast(encodeEnvelope(&Envelope{Type: MsgTypePush, MsgId: msgId, Payload: data}))
	return nil
}
