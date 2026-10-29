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

var _ btnet.InHandler = (*RouterHandler)(nil)

type route struct {
	method string
	public bool // 免登录
	decode func(payload []byte) (any, error)
	call   func(ctx *RequestContext, req any) (any, error) // 业务方法
}

type RouterHandler struct {
	codec        Codec
	routes       map[uint32]*route
	interceptors []Interceptor
	frozen       atomic.Bool
}

func NewRouterHandler(codec Codec) *RouterHandler {
	if codec == nil {
		codec = JSONCodec{}
	}
	return &RouterHandler{codec: codec, routes: make(map[uint32]*route)}
}

func (r *RouterHandler) AddInterceptor(its ...Interceptor) {
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
	RegisterRoutes(r *RouterHandler)
}

func (r *RouterHandler) RegisterModules(ms ...Module) {
	for _, m := range ms {
		m.RegisterRoutes(r)
	}
}

func Handle[Req any, Resp any](r *RouterHandler, msgId uint32, fn func(*RequestContext, *Req) (*Resp, error)) {
	r.addRoute(msgId, newRoute(r, msgId, fn, false))
}

// HandlePublic 注册免登录路由（登录、心跳）
func HandlePublic[Req any, Resp any](r *RouterHandler, msgId uint32, fn func(*RequestContext, *Req) (*Resp, error)) {
	r.addRoute(msgId, newRoute(r, msgId, fn, true))
}

func newRoute[Req any, Resp any](r *RouterHandler, msgId uint32, fn func(*RequestContext, *Req) (*Resp, error), public bool) *route {
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

func (r *RouterHandler) unmarshal(payload []byte, req any) error {
	if len(payload) == 0 { // 无参请求可不带 payload
		return nil
	}
	return r.codec.Unmarshal(payload, req)
}

func (r *RouterHandler) addRoute(msgId uint32, rt *route) {
	if r.frozen.Load() {
		panic(fmt.Sprintf("register msgId %d after dispatch started", msgId))
	}
	if old, ok := r.routes[msgId]; ok {
		panic(fmt.Sprintf("msgId %d already registered by %s", msgId, old.method))
	}
	r.routes[msgId] = rt
}

func (r *RouterHandler) freeze() {
	if !r.frozen.Load() {
		r.frozen.Store(true)
	}
}

func funcName(fn any) string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	name = name[strings.LastIndexByte(name, '.')+1:]
	return strings.TrimSuffix(name, "-fm")
}

func (r *RouterHandler) OnRead(ctx *btnet.HandlerContext, msg any) {
	env, ok := msg.(*Envelope)
	if !ok {
		ctx.FireRead(msg)
		return
	}
	r.dispatch(ctx.GetConnection(), env)
}

func (r *RouterHandler) dispatch(conn *btnet.Connection, env *Envelope) {
	r.freeze()
	if env.Type != MsgTypeRequest {
		btlog.Ins().Warn("unexpectedMsgType", "connId", conn.GetId(), "type", env.Type, "msgId", env.MsgId)
		return
	}

	rt := r.routes[env.MsgId]
	if rt == nil {
		btlog.Ins().Warn("noRoute", "connId", conn.GetId(), "msgId", env.MsgId)
		r.reply(conn, env, nil, CodeUnimplemented)
		return
	}

	ctx := &RequestContext{Context: conn.Context(), conn: conn, msgId: env.MsgId, route: rt, startAt: time.Now()}
	resp, err := r.invoke(ctx, rt, env.Payload)
	r.reply(conn, env, resp, err)
}

func (r *RouterHandler) invoke(ctx *RequestContext, rt *route, payload []byte) (resp any, err error) {
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
		btlog.Ins().Debug("invoke", "connId", ctx.conn.GetId(), "msgId", ctx.msgId, "method", rt.method)
	}
	for i := n - 1; i >= 0; i-- {
		r.safeAfter(ctx, rt, r.interceptors[i], req, resp, err)
	}
	return resp, err
}

func (r *RouterHandler) safeDecode(ctx *RequestContext, rt *route, payload []byte) (req any, err error) {
	defer r.recoverPanic(ctx, rt, "decode", nil, &err)
	if req, err = rt.decode(payload); err != nil {
		btlog.Ins().Warn("decodeError", "connId", ctx.conn.GetId(), "msgId", ctx.msgId, "method", rt.method, "err", err)
		return nil, CodeInvalidArgument
	}
	return req, nil
}

func (r *RouterHandler) safeBefore(ctx *RequestContext, rt *route, it Interceptor, req any) (err error) {
	defer r.recoverPanic(ctx, rt, "interceptor.Before", it, &err)
	return it.Before(ctx, req)
}

func (r *RouterHandler) safeCall(ctx *RequestContext, rt *route, req any) (resp any, err error) {
	defer r.recoverPanic(ctx, rt, "handler", nil, &err)
	return rt.call(ctx, req)
}

func (r *RouterHandler) safeAfter(ctx *RequestContext, rt *route, it Interceptor, req, resp any, err error) {
	var ignored error
	defer r.recoverPanic(ctx, rt, "interceptor.After", it, &ignored)
	it.After(ctx, req, resp, err)
}

func (r *RouterHandler) recoverPanic(ctx *RequestContext, rt *route, stage string, it Interceptor, err *error) {
	if p := recover(); p != nil {
		if it != nil {
			stage = fmt.Sprintf("%s(%T)", stage, it)
		}
		btlog.Ins().Error("handlerPanic", "stage", stage, "connId", ctx.conn.GetId(), "msgId", ctx.msgId,
			"method", rt.method, "panic", p, "stack", string(debug.Stack()))
		*err = CodeInternal
	}
}

func (r *RouterHandler) reply(conn *btnet.Connection, req *Envelope, resp any, err error) {
	code := r.toCode(conn, req, err)
	if req.Seq == 0 {
		return
	}

	out := &Envelope{Type: MsgTypeResponse, Seq: req.Seq, MsgId: req.MsgId, Code: uint32(code)}
	if code == CodeOK && resp != nil {
		b, mErr := r.codec.Marshal(resp)
		if mErr != nil {
			btlog.Ins().Error("marshalRespError", "connId", conn.GetId(), "msgId", req.MsgId, "err", mErr)
			out.Code = uint32(CodeInternal)
		} else {
			out.Payload = b
		}
	}
	_ = conn.Write(out)
}

func (r *RouterHandler) toCode(conn *btnet.Connection, env *Envelope, err error) Code {
	if err == nil {
		return CodeOK
	}
	var code Code
	if errors.As(err, &code) {
		// 包装过的 Code 带有上下文（如底层 DB 错误），日志打完整链，客户端只拿到 code
		if err != error(code) {
			btlog.Ins().Warn("codeError", "connId", conn.GetId(), "msgId", env.MsgId, "code", uint32(code), "err", err)
		}
		return code
	}
	btlog.Ins().Error("nonCodeError", "connId", conn.GetId(), "msgId", env.MsgId, "err", err)
	return CodeInternal
}

func (r *RouterHandler) Push(conn *btnet.Connection, msgId uint32, v any) error {
	data, err := r.codec.Marshal(v)
	if err != nil {
		return err
	}
	return conn.Write(&Envelope{Type: MsgTypePush, MsgId: msgId, Payload: data})
}

func (r *RouterHandler) Broadcast(mgr *btnet.ConnManager, msgId uint32, v any) error {
	data, err := r.codec.Marshal(v)
	if err != nil {
		return err
	}
	// 编码一次
	mgr.Broadcast(encodeEnvelope(&Envelope{Type: MsgTypePush, MsgId: msgId, Payload: data}))
	return nil
}
