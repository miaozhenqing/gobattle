package service

import (
	"context"
	"gobattle/btlog"
	"runtime/debug"
	"sync"
	"time"
)

type tickLoop struct {
	name     string
	interval time.Duration
	fn       func()
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func newTickLoop(name string, interval time.Duration, fn func()) *tickLoop {
	return &tickLoop{name: name, interval: interval, fn: fn}
}

func (l *tickLoop) run() {
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		ticker := time.NewTicker(l.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				l.safeTick()
			}
		}
	}()
}

func (l *tickLoop) stop() {
	if l.cancel != nil {
		l.cancel()
	}
	l.wg.Wait()
}

func (l *tickLoop) safeTick() {
	defer func() {
		if r := recover(); r != nil {
			btlog.Ins().Error("tickPanic", "tick", l.name, "panic", r, "stack", string(debug.Stack()))
		}
	}()
	l.fn()
}
