package btnet

import (
	"context"
	"fmt"
	"gobattle/btiface"
)

type EventLoopGroup struct {
	loops []*EventLoop
}

func NewEventLoopGroup(name string, size int) *EventLoopGroup {
	if size <= 0 {
		panic(fmt.Sprintf("eventloop group %s size must be > 0", name))
	}
	g := &EventLoopGroup{loops: make([]*EventLoop, size)}
	for i := range g.loops {
		g.loops[i] = NewEventLoop(fmt.Sprintf("%s-%d", name, i))
		go g.loops[i].run()
	}
	return g
}

func (g *EventLoopGroup) Next(key int64) btiface.IEventLoop {
	return g.loops[uint64(key)%uint64(len(g.loops))]
}

func (g *EventLoopGroup) ShutdownGracefully(ctx context.Context) error {
	//先通知loop停止，不再接收task
	for _, loop := range g.loops {
		loop.shutdown()
	}
	//逐个等待退出
	for _, loop := range g.loops {
		select {
		case <-loop.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
