package btnet

import (
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
