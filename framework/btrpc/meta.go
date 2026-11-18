package btrpc

import (
	"context"
	"strconv"
)

type RequestMeta struct {
	UserId   int64
	PeerType string
	PeerId   int64
}

type requestMetaKey struct{}
type hashKeyKey struct{}

func WithRequestMeta(ctx context.Context, m RequestMeta) context.Context {
	return context.WithValue(ctx, requestMetaKey{}, m)
}

func GetRequestMeta(ctx context.Context) RequestMeta {
	m, _ := ctx.Value(requestMetaKey{}).(RequestMeta)
	return m
}

func UserId(ctx context.Context) int64 { return GetRequestMeta(ctx).UserId }

func WithHashKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, hashKeyKey{}, key)
}

func HashKey(ctx context.Context) string {
	if k, ok := ctx.Value(hashKeyKey{}).(string); ok {
		return k
	}
	return strconv.FormatInt(UserId(ctx), 10)
}
