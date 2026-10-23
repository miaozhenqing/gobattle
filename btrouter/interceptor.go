package btrouter

import (
	"gobattle/btlog"
	"gobattle/btnet"
	"slices"
	"time"

	"golang.org/x/time/rate"
)

// Interceptor 按 AddInterceptor 顺序执行 Before，Before 返回 err 即中断；
// 之后对 Before 已成功的拦截器逆序执行 After
type Interceptor interface {
	Before(ctx *Context, req any) error
	After(ctx *Context, req, resp any, err error)
}

// 按连接限流
func RateLimit(perSec float64, burst int) Interceptor {
	if perSec <= 0 || burst <= 0 {
		panic("perSec and burst must be > 0")
	}
	return &rateLimit{perSec: perSec, burst: burst, key: btnet.NewAttrKey[*rate.Limiter]("btrouter.rateLimit")}
}

type rateLimit struct {
	perSec float64
	burst  int
	key    *btnet.AttrKey[*rate.Limiter]
}

func (l *rateLimit) Before(ctx *Context, req any) error {
	lim, ok := l.key.Get(ctx.conn)
	if !ok {
		lim = rate.NewLimiter(rate.Limit(l.perSec), l.burst)
		l.key.Set(ctx.conn, lim)
	}
	if !lim.Allow() {
		return errResourceExhausted
	}
	return nil
}

func (l *rateLimit) After(ctx *Context, req, resp any, err error) {}

// Auth skip 中的 msgId 不校验
func Auth(authed func(*Context) bool, skip ...uint32) Interceptor {
	if authed == nil {
		panic("authed is nil")
	}
	return &auth{authed: authed, skip: slices.Clone(skip)}
}

type auth struct {
	authed func(*Context) bool
	skip   []uint32
}

func (a *auth) Before(ctx *Context, req any) error {
	if !slices.Contains(a.skip, ctx.msgId) && !a.authed(ctx) {
		return errUnauthenticated
	}
	return nil
}

func (a *auth) After(ctx *Context, req, resp any, err error) {}

// SlowLog 从 dispatch 收到请求开始计时
func SlowLog(threshold time.Duration) Interceptor {
	if threshold <= 0 {
		panic("threshold must be > 0")
	}
	return &slowLog{threshold: threshold}
}

type slowLog struct {
	threshold time.Duration
}

func (s *slowLog) Before(ctx *Context, req any) error { return nil }

func (s *slowLog) After(ctx *Context, req, resp any, err error) {
	if cost := time.Since(ctx.startAt); cost >= s.threshold {
		btlog.Ins().Warnf("slow request, connID:%v msgId:%v(%s) cost:%v",
			ctx.conn.GetID(), ctx.msgId, ctx.method, cost)
	}
}
