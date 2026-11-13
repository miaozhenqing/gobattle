package module

import (
	"context"
	"gobattle/framework/btrouter"
	"gobattle/framework/btrpc"
	"gobattle/framework/btsession"
)

func rpcCtx(ctx *btrouter.RequestContext) context.Context {
	sess, _ := btsession.GetSession(ctx.Conn())
	return btrpc.WithRequestMeta(ctx, btrpc.RequestMeta{UserId: sess.UserId()})
}
