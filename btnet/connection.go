package btnet

import (
	"context"
	"errors"
	"gobattle/btlog"
	"io"
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const (
	connActive int32 = iota
	connClosing
	connClosed
)

type Connection struct {
	id           int64
	conn         *net.TCPConn
	opts         *Options
	msgChan      chan []byte
	frameDecoder *FrameDecoder
	connMgr      *ConnManager
	cancelCtx    context.Context
	cancelFunc   context.CancelFunc
	pipeline     *Pipeline
	stopOnce     sync.Once
	state        atomic.Int32
	flushChan    chan struct{}

	ioLoop *EventLoop

	attMu sync.RWMutex
	attrs map[any]any

	onInactiveDone   func()
	inactiveDoneOnce sync.Once
}

func NewConnection(id int64, conn *net.TCPConn, opts *Options, ioLoop *EventLoop, connMgr *ConnManager,
	initializer PipelineInitializer) (*Connection, error) {
	c := &Connection{
		id:      id,
		conn:    conn,
		opts:    opts,
		ioLoop:  ioLoop,
		msgChan: make(chan []byte, opts.SendQueueSize),
		// 解码器大概不会变，直接内部创建一个
		frameDecoder: NewFrameDecoder(opts.MaxPacketSize, opts.IOReadBuffSize),
		connMgr:      connMgr,
		flushChan:    make(chan struct{}),
	}
	c.cancelCtx, c.cancelFunc = context.WithCancel(context.Background())
	p := newPipeline(c, ioLoop)
	if err := runInitializer(initializer, p); err != nil {
		c.cancelFunc()
		return nil, err
	}
	c.pipeline = p
	return c, nil
}

func runInitializer(init PipelineInitializer, p *Pipeline) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &PanicError{Value: r, Stack: debug.Stack()}
		}
	}()
	init(p)
	return nil
}

func (c *Connection) GetId() int64 {
	return c.id
}
func (c *Connection) Start() {
	go c.startRead()
	go c.startWrite()
	//先active，再read
	c.pipeline.fireActiveInLoop()
}

func (c *Connection) Stop() {
	c.stopOnce.Do(func() {
		c.state.Store(connClosed)
		c.cancelFunc()
		c.destroy()
		c.ioLoop.Execute(c.pipeline.fireInactiveInLoop)
	})
}

func (c *Connection) FlushAndStop() {
	if c.state.CompareAndSwap(connActive, connClosing) {
		c.ioLoop.Execute(func() {
			close(c.flushChan)
		})
	}
}
func (c *Connection) IsClosed() bool {
	return c.state.Load() != connActive
}

func (c *Connection) destroy() {
	c.conn.Close()
	c.connMgr.Remove(c.GetId())
}

func (c *Connection) Write(msg any) error {
	if c.IsClosed() {
		return ErrConnClosed
	}
	c.pipeline.FireWrite(msg)
	return nil
}

func (c *Connection) Context() context.Context {
	return c.cancelCtx
}

func (c *Connection) writeRaw(data []byte) {
	select {
	case c.msgChan <- data:
	case <-c.cancelCtx.Done():
	default:
		btlog.Ins().Error("sendQueueFull", "connId", c.GetId())
		c.Stop()
	}
}

func (c *Connection) startRead() {
	defer func() {
		if c.state.Load() != connClosing {
			c.Stop()
		}
	}()
	buf := make([]byte, c.opts.IOReadBuffSize)
	// 上次解出完整帧的时间
	activeAt := time.Now()

	for {
		if err := c.conn.SetReadDeadline(c.readDeadline(activeAt)); err != nil {
			btlog.Ins().Error("setReadDeadlineError", "connId", c.GetId(), "err", err)
			return
		}
		//读取数据到数组
		size, err := c.conn.Read(buf)
		if err != nil {
			var ne net.Error
			if c.IsClosed() {
				btlog.Ins().Info("readDone", "connId", c.GetId())
			} else if errors.Is(err, io.EOF) {
				btlog.Ins().Info("peerClosed", "connId", c.GetId())
			} else if errors.As(err, &ne) && ne.Timeout() {
				btlog.Ins().Info("idleTimeout", "connId", c.GetId())
			} else {
				c.fireException(err)
			}
			return
		}
		if size == 0 {
			continue
		}
		//解包
		frames, err := c.frameDecoder.Decode(buf[:size])
		if err != nil {
			c.fireException(err)
			return
		}
		if len(frames) == 0 {
			continue
		}
		activeAt = time.Now()
		//解码
		c.ioLoop.Execute(func() {
			for _, f := range frames {
				if c.IsClosed() {
					return
				}
				c.pipeline.fireReadInLoop(f)
			}
		})
	}
}

func (c *Connection) readDeadline(activeAt time.Time) time.Time {
	if t := c.opts.IdleTimeout; t > 0 {
		return activeAt.Add(t)
	}
	return time.Time{}
}

func (c *Connection) startWrite() {
	defer c.Stop()
	for {
		select {
		case data := <-c.msgChan:
			if err := c.write(data); err != nil {
				return
			}
		case <-c.cancelCtx.Done():
			btlog.Ins().Info("writeDone", "connId", c.GetId())
			return
		case <-c.flushChan:
			c.flush()
			return
		}
	}
}

func (c *Connection) flush() {
	var deadline time.Time
	if t := c.opts.FlushTimeout; t > 0 {
		deadline = time.Now().Add(t)
	}
	for {
		select {
		case data := <-c.msgChan:
			if err := c.writeWithDeadline(data, deadline); err != nil {
				return
			}
		default:
			return
		}
	}
}

func (c *Connection) writeWithDeadline(data []byte, deadline time.Time) error {
	if err := c.conn.SetWriteDeadline(deadline); err != nil {
		btlog.Ins().Error("setWriteDeadlineError", "connId", c.GetId(), "err", err)
		return err
	}
	if _, err := c.conn.Write(data); err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			btlog.Ins().Error("writeTimeout", "connId", c.GetId())
		} else {
			btlog.Ins().Error("writeError", "connId", c.GetId(), "err", err)
		}
		return err
	}
	return nil
}
func (c *Connection) write(data []byte) error {
	var deadline time.Time
	if t := c.opts.WriteTimeout; t > 0 {
		deadline = time.Now().Add(t)
	}
	return c.writeWithDeadline(data, deadline)
}

func (c *Connection) EventLoop() *EventLoop {
	return c.ioLoop
}

func (c *Connection) GetPipeline() *Pipeline {
	return c.pipeline
}

func (c *Connection) fireException(err error) {
	c.ioLoop.Execute(func() {
		c.pipeline.fireExceptionInLoop(err)
	})
}

func (c *Connection) inactiveDone() {
	c.inactiveDoneOnce.Do(func() {
		if c.onInactiveDone != nil {
			c.onInactiveDone()
		}
	})
}
