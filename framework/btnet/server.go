package btnet

import (
	"context"
	"errors"
	"gobattle/framework/btlog"
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
	workerGroup *EventLoopGroup
}

func NewServer(opts Options, worker *EventLoopGroup) *Server {
	s := &Server{
		opts:        opts,
		connMgr:     NewConnManager(),
		workerGroup: worker,
	}
	return s
}

func (s *Server) SetPipelineInitializer(fn PipelineInitializer) {
	s.initializer = fn
}

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
	btlog.Ins().Info("serverListening", "server", s.opts.Name, "addr", listener.Addr().String(), "maxConn", s.opts.MaxConn,
		"maxPacketSize", s.opts.MaxPacketSize, "idleTimeout", s.opts.IdleTimeout.String(), "writeTimeout", s.opts.WriteTimeout.String())
	go s.acceptLoop()
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
			btlog.Ins().Warn("overMaxConn", "server", s.opts.Name, "maxConn", s.opts.MaxConn)
			continue
		}

		tcp, err := s.listener.AcceptTCP()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				//被关闭，永久退出
				btlog.Ins().Info("listenerClosed", "server", s.opts.Name, "err", err)
				return
			}

			// 其他,退避重试避免空转todo 后续可拆出来优化
			if delay == 0 {
				delay = minAcceptDelay
			} else if delay *= 2; delay > maxAcceptDelay {
				delay = maxAcceptDelay
			}
			btlog.Ins().Error("acceptError", "server", s.opts.Name, "err", err, "retryMs", delay.Milliseconds())
			time.Sleep(delay)
			continue
		}
		delay = 0

		connId := generalConnId()
		loop := s.workerGroup.Next(connId)
		loop.Execute(func() {
			connection, err2 := NewConnection(connId, tcp, &s.opts, loop, s.connMgr, s.initializer)
			if err2 != nil {
				btlog.Ins().Error("newConnectionError", "server", s.opts.Name, "connId", connId, "remote", tcp.RemoteAddr().String(), "err", err2)
				tcp.Close()
				return
			}
			connection.onInactiveDone = s.connMgr.done

			if !s.connMgr.Add(connection) {
				btlog.Ins().Info("serverClosingRejectConn", "server", s.opts.Name, "connId", connId)
				tcp.Close()
				return
			}
			btlog.Ins().Info("connAccepted", "server", s.opts.Name, "connId", connId, "remote", tcp.RemoteAddr().String(),
				"conns", s.connMgr.Len())
			connection.Start()
		})
	}
}

var connIdCounter atomic.Int64

func generalConnId() int64 {
	return connIdCounter.Add(1)
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
	s.connMgr.markClosed()
	conns := s.connMgr.snapshot()
	btlog.Ins().Info("serverShutdown", "server", s.opts.Name, "conns", len(conns))
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
		btlog.Ins().Info("serverShutdownDone", "server", s.opts.Name)
		return nil
	case <-ctx.Done():
		// 超时强停
		btlog.Ins().Error("shutdownTimeoutForceStop", "server", s.opts.Name, "conns", len(conns))
		for _, c := range conns {
			c.Stop()
		}
		return ctx.Err()
	}

}
