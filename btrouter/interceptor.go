package btrouter

import (
	"gobattle/btlog"
	"gobattle/btnet"
	"time"

	"golang.org/x/time/rate"
)

type Interceptor interface {
	Before(ctx *RequestContext, req any) error
	After(ctx *RequestContext, req, resp any, err error)
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

func (l *rateLimit) Before(ctx *RequestContext, req any) error {
	lim, ok := l.key.Get(ctx.conn)
	if !ok {
		lim = rate.NewLimiter(rate.Limit(l.perSec), l.burst)
		l.key.Set(ctx.conn, lim)
	}
	if !lim.Allow() {
		return CodeResourceExhausted
	}
	return nil
}

func (l *rateLimit) After(ctx *RequestContext, req, resp any, err error) {}

func Auth(authed func(*RequestContext) bool) Interceptor {
	if authed == nil {
		panic("authed is nil")
	}
	return &auth{authed: authed}
}

type auth struct {
	authed func(*RequestContext) bool
}

func (a *auth) Before(ctx *RequestContext, req any) error {
	if ctx.IsPublic() {
		return nil
	}
	if !a.authed(ctx) {
		return CodeUnauthenticated
	}
	return nil
}

func (a *auth) After(ctx *RequestContext, req, resp any, err error) {}

func SlowLog(threshold time.Duration) Interceptor {
	if threshold <= 0 {
		panic("threshold must be > 0")
	}
	return &slowLog{threshold: threshold}
}

type slowLog struct {
	threshold time.Duration
}

func (s *slowLog) Before(ctx *RequestContext, req any) error { return nil }

func (s *slowLog) After(ctx *RequestContext, req, resp any, err error) {
	if cost := time.Since(ctx.startAt); cost >= s.threshold {
		btlog.Ins().Warnf("slow request, connID:%v msgId:%v(%s) cost:%v",
			ctx.conn.GetID(), ctx.msgId, ctx.Method(), cost)
	}
}
