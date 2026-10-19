package btnet

import (
	"context"
	"errors"
	"gobattle/btiface"
	"gobattle/btlog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var ErrConnClosed = errors.New("conn is closed")

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
	frameDecoder btiface.IFrameDecoder
	connMgr      btiface.IConnManager
	cancelCtx    context.Context
	cancelFunc   context.CancelFunc
	pipeline     *Pipeline
	stopOnce     sync.Once
	state        atomic.Int32
	flushChan    chan struct{}

	ioLoop btiface.IEventLoop
}

func NewConnection(id int64, conn *net.TCPConn, opts *Options, ioLoop btiface.IEventLoop, connMgr btiface.IConnManager,
	initializer btiface.PipelineInitializer) *Connection {
	c := &Connection{
		id:           id,
		conn:         conn,
		opts:         opts,
		ioLoop:       ioLoop,
		msgChan:      make(chan []byte, opts.SendQueueSize),
		frameDecoder: NewFrameDecoder(opts.MaxPacketSize, opts.IOReadBuffSize),
		connMgr:      connMgr,
		flushChan:    make(chan struct{}),
	}
	c.cancelCtx, c.cancelFunc = context.WithCancel(context.Background())
	p := NewPipeline(c, ioLoop)
	initializer(p)
	c.pipeline = p
	return c
}

func (c *Connection) GetID() int64 {
	return c.id
}
func (c *Connection) Start() {
	go c.startRead()
	go c.startWrite()
}

func (c *Connection) Stop() {
	c.stopOnce.Do(func() {
		c.state.Store(connClosed)
		c.cancelFunc()
		c.destroy()
	})
}

func (c *Connection) FlushAndStop() {
	if c.state.CompareAndSwap(connActive, connClosing) {
		close(c.flushChan)
	}
}
func (c *Connection) IsClosed() bool {
	return c.state.Load() != connActive
}

func (c *Connection) destroy() {
	c.conn.Close()
	c.connMgr.Remove(c.GetID())
}

func (c *Connection) SendMsg(msgId uint32, data []byte) error {
	if c.IsClosed() {
		return ErrConnClosed
	}
	c.pipeline.FireWrite(NewMessage(msgId, data))
	return nil
}

func (c *Connection) writeRaw(data []byte) {
	select {
	case c.msgChan <- data:
	case <-c.cancelCtx.Done():
	default:
		btlog.Ins().Errorf("send queue full, connID:%v", c.GetID())
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
			btlog.Ins().Errorf("set read deadline error, connID:%v, err:%v", c.GetID(), err)
			return
		}
		//读取数据到数组
		size, err := c.conn.Read(buf)
		if err != nil {
			var ne net.Error
			if c.IsClosed() {
				btlog.Ins().Infof("read done, connID:%v", c.GetID())
			} else if errors.As(err, &ne) && ne.Timeout() {
				btlog.Ins().Infof("idle timeout, connID:%v", c.GetID())
			} else {
				btlog.Ins().Errorf("read error, connID:%v, err:%v", c.GetID(), err)
			}
			return
		}
		if size == 0 {
			continue
		}
		//解包
		msgs, err := c.frameDecoder.Decode(buf[:size])
		if err != nil {
			btlog.Ins().Errorf("decode error:%v", err)
			return
		}
		if len(msgs) == 0 {
			continue
		}
		activeAt = time.Now()
		//解码
		c.ioLoop.Execute(func() {
			for _, m := range msgs {
				if c.IsClosed() {
					return
				}
				c.pipeline.fireReadInLoop(m)
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
			btlog.Ins().Infof("write done, connID:%v", c.GetID())
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
		btlog.Ins().Errorf("set write deadline error, connID:%v, err:%v", c.GetID(), err)
		return err
	}
	if _, err := c.conn.Write(data); err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			btlog.Ins().Errorf("write timeout, connID:%v", c.GetID())
		} else {
			btlog.Ins().Errorf("write error, connID:%v, error:%v", c.GetID(), err)
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

func (c *Connection) EventLoop() btiface.IEventLoop {
	return c.ioLoop
}

func (c *Connection) GetPipeline() btiface.IPipeline {
	return c.pipeline
}
