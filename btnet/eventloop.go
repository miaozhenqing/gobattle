package btnet

import (
	"context"
	"fmt"
	"gobattle/btlog"
	"runtime/debug"
	"sync"
)

type EventLoop struct {
	name   string
	mu     sync.Mutex
	tasks  []func()
	notify chan struct{}

	stopped bool
	done    chan struct{}
}

func NewEventLoop(name string) *EventLoop {
	return &EventLoop{
		name:   name,
		notify: make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
}

func (el *EventLoop) Execute(task func()) {
	el.mu.Lock()
	defer el.mu.Unlock()
	if el.stopped {
		btlog.Ins().Warnf("eventloop stopped, task dropped, loop:%v", el.name)
		return
	}
	el.tasks = append(el.tasks, task)
	select {
	//有任务进来，唤醒
	case el.notify <- struct{}{}:
	default:

	}
}

func (el *EventLoop) shutdown() {
	el.mu.Lock()
	defer el.mu.Unlock()
	if !el.stopped {
		el.stopped = true
		close(el.notify)
	}
}

func (el *EventLoop) run() {
	defer close(el.done)
	var batch []func()
	for range el.notify {
		el.mu.Lock()
		batch, el.tasks = el.tasks, batch[:0]
		el.mu.Unlock()
		for i, t := range batch {
			el.safeRun(t)
			batch[i] = nil
		}
	}
}

func (el *EventLoop) safeRun(task func()) {
	defer func() {
		if r := recover(); r != nil {
			btlog.Ins().Errorf("eventloop panic, loop:%v err:%v  %s", el.name, r, debug.Stack())
		}
	}()
	task()
}

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

func (g *EventLoopGroup) Next(key int64) *EventLoop {
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
