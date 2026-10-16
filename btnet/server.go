package btnet

import (
	"errors"
	"gobattle/btiface"
	"gobattle/btlog"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

type server struct {
	opts           Options
	connMgr        btiface.IConnManager
	messageHandler btiface.IMessageHandler
	initializer    btiface.PipelineInitializer
	listener       *net.TCPListener
}

func NewServer(opts Options) *server {
	s := &server{
		opts:           opts,
		connMgr:        NewConnManager(),
		messageHandler: NewMessageHandler(opts.WorkerPoolSize, opts.WorkerPoolQueueSize),
	}
	return s
}

func (s *server) SetPipelineInitializer(fn btiface.PipelineInitializer) {
	s.initializer = fn
}

func (s *server) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
	s.connMgr.CloseAll()
}

func (s *server) Start() error {
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
	go s.acceptLoop()
	return nil
}

var (
	minAcceptDelay = 5 * time.Millisecond
	maxAcceptDelay = time.Second
)

func (s *server) acceptLoop() {
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

			// 其他,退避重试避免空转
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

		go func() {
			connID := generalConnID()
			connection := NewConnection(connID, tcp, &s.opts, s.messageHandler, s.connMgr, s.initializer)
			s.connMgr.Add(connection)
			connection.Start()
		}()
	}
}

var connIDCounter atomic.Int64

func generalConnID() int64 {
	return connIDCounter.Add(1)
}
