package btnet

import (
	"context"
	"errors"
	"gobattle/btlog"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

type Server struct {
	opts        Options
	connMgr     *ConnManager
	initializer PipelineInitializer
	listener    *net.TCPListener
	bossGroup   *EventLoopGroup
	workerGroup *EventLoopGroup
}

func NewServer(opts Options, boss, worker *EventLoopGroup) *Server {
	s := &Server{
		opts:        opts,
		connMgr:     NewConnManager(),
		bossGroup:   boss,
		workerGroup: worker,
	}
	return s
}

func (s *Server) SetPipelineInitializer(fn PipelineInitializer) {
	s.initializer = fn
}

// Stop 强制关闭，不排空发送队列；优雅关闭用 Shutdown
func (s *Server) Stop() {
	s.closeListener()
	s.connMgr.CloseAll()
}

func (s *Server) Start() error {
	if err := s.opts.validate(); err != nil {
		return err
	}
	if s.initializer == nil {
		return errors.New("pipeline initializer not set")
	}
	addr, err := net.ResolveTCPAddr(s.opts.IPVersion, s.opts.Host+":"+strconv.Itoa(s.opts.TcpPort))
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP(s.opts.IPVersion, addr)
	if err != nil {
		return err
	}
	s.listener = listener
	s.bossGroup.Next(0).Execute(s.acceptLoop)
	return nil
}

var (
	minAcceptDelay = 5 * time.Millisecond
	maxAcceptDelay = time.Second
)

func (s *Server) acceptLoop() {
	var delay time.Duration
	for {
		if s.connMgr.Len() >= s.opts.MaxConn {
			//todo 先暴力sleep
			time.Sleep(1000 * time.Millisecond)
			btlog.Ins().Warn("over max conn")
			continue
		}

		tcp, err := s.listener.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				//被关闭，永久退出
				btlog.Ins().Infof("listener closed, exit loop:%v", err)
				return
			}

			// 其他,退避重试避免空转todo 后续可拆出来优化
			if delay == 0 {
				delay = minAcceptDelay
			} else if delay *= 2; delay > maxAcceptDelay {
				delay = maxAcceptDelay
			}
			btlog.Ins().Errorf("accept tcp error:%v, retry in %v", err, delay)
			time.Sleep(delay)
			continue
		}
		delay = 0

		connID := generalConnID()
		loop := s.workerGroup.Next(connID)
		loop.Execute(func() {
			connection, err2 := NewConnection(connID, tcp, &s.opts, loop, s.connMgr, s.initializer)
			if err2 != nil {
				btlog.Ins().Errorf("new connection error, connID:%v, remote:%v, err:%v", connID, tcp.RemoteAddr(), err2)
				tcp.Close()
				return
			}
			connection.onInactiveDone = s.connMgr.done

			if !s.connMgr.Add(connection) {
				btlog.Ins().Infof("server closing, reject conn, connID:%v", connID)
				tcp.Close()
				return
			}
			connection.Start()
		})
	}
}

var connIDCounter atomic.Int64

func generalConnID() int64 {
	return connIDCounter.Add(1)
}

func (s *Server) closeListener() {
	if s.listener != nil {
		_ = s.listener.Close()
	}
}

func (s *Server) ConnManager() *ConnManager {
	return s.connMgr
}
func (s *Server) Shutdown(ctx context.Context) error {
	s.closeListener()
	conns := s.connMgr.close()
	for _, c := range conns {
		c.FlushAndStop()
	}
	done := make(chan struct{})
	go func() {
		s.connMgr.wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		btlog.Ins().Errorf("shutdown timeout, force stop, conns:%v, maybe some OnInactive not propagated", len(conns))
		for _, c := range conns {
			c.Stop()
		}
		return ctx.Err()
	}

}
