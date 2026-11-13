package btnet

import (
	"context"
	"gobattle/framework/btlog"
)

type ServerGroup struct {
	Name   string
	Server *Server
	Groups []*EventLoopGroup
}

func (g *ServerGroup) Shutdown(ctx context.Context) {
	if err := g.Server.Shutdown(ctx); err != nil {
		btlog.Ins().Error("serverShutdownError", "server", g.Name, "err", err)
	}
	for _, eg := range g.Groups {
		if err := eg.ShutdownGracefully(ctx); err != nil {
			btlog.Ins().Error("eventloopShutdownError", "server", g.Name, "err", err)
		}
	}
}
