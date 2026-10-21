package btnet

import (
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
