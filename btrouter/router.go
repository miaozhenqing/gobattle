package btrouter

import (
	"fmt"
	"gobattle/btiface"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
)

var _ btiface.IDispatcher = (*Router)(nil)

type handlerFunc func(ctx *Context, payload []byte) (resp any, err error)

type route struct {
	method string
	h      handlerFunc
	oneway bool
}

type Router struct {
	codec  Codec
	routes map[uint32]*route
	frozen atomic.Bool
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
	r.add(msgId, fn, false, func(ctx *Context, payload []byte) (any, error) {
		req := new(Req)
		if err := r.unmarshal(payload, req); err != nil {
			return nil, err
		}
		resp, err := fn(ctx, req)
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
	r.add(msgId, fn, true, func(ctx *Context, payload []byte) (any, error) {
		req := new(Req)
		if err := r.unmarshal(payload, req); err != nil {
			return nil, err
		}
		return nil, fn(ctx, req)
	})
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

func (r *Router) add(msgId uint32, fn any, oneway bool, h handlerFunc) {
	if r.frozen.Load() {
		panic(fmt.Sprintf("register msgId %d after dispatch started", msgId))
	}
	if msgId < MinRequestMsgId || msgId >= MinPushMsgId {
		panic(fmt.Sprintf("request msgId %d out of range [%d, %d)", msgId, MinRequestMsgId, MinPushMsgId))
	}
	if old, ok := r.routes[msgId]; ok {
		panic(fmt.Sprintf("msgId %d already registered by %s", msgId, old.method))
	}
	r.routes[msgId] = &route{method: funcName(fn), h: h, oneway: oneway}
}

func funcName(fn any) string {
	name := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	return strings.TrimSuffix(name, "-fm")
}
