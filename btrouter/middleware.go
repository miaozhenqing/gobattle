package btrouter

import (
	"gobattle/btiface"
	"gobattle/btlog"
	"slices"
	"time"

	"golang.org/x/time/rate"
)

func SlowLog(threshold time.Duration) Middleware {
	return func(ctx *Context, req any, next Handler) (any, error) {
		start := time.Now()
		resp, err := next(ctx, req)
		if cost := time.Since(start); cost >= threshold {
			btlog.Ins().Warnf("slow request, connID:%v msgId:%v(%s) cost:%v",
				ctx.conn.GetID(), ctx.msgId, ctx.method, cost)
		}
		return resp, err
	}
}

func Auth(authed func(*Context) bool, skip ...uint32) Middleware {
	if authed == nil {
		panic("authed is nil")
	}
	skip = slices.Clone(skip)
	return func(ctx *Context, req any, next Handler) (any, error) {
		if !slices.Contains(skip, ctx.msgId) && !authed(ctx) {
			return nil, errUnauthenticated
		}
		return next(ctx, req)
	}
}

func RateLimit(perSec float64, burst int) Middleware {
	if perSec <= 0 || burst <= 0 {
		panic("perSec and burst must be > 0")
	}
	key := btiface.NewAttrKey[*rate.Limiter]("btrouter.rateLimit")
	return func(ctx *Context, req any, next Handler) (any, error) {
		l, ok := key.Get(ctx.conn)
		if !ok {
			l = rate.NewLimiter(rate.Limit(perSec), burst)
			key.Set(ctx.conn, l)
		}
		if !l.Allow() {
			return nil, errResourceExhausted
		}
		return next(ctx, req)
	}
}
