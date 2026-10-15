package btnet

import (
	"errors"
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/config"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

type server struct {
	name           string
	ip             string
	port           int
	IPVersion      string
	connMgr        btiface.IConnManager
	messageHandler btiface.IMessageHandler
	initializer    btiface.PipelineInitializer
	listener       *net.TCPListener
}

func NewServer() *server {
	s := &server{
		name:           config.ServerConfig.Name,
		ip:             config.ServerConfig.Host,
		port:           config.ServerConfig.TcpPort,
		IPVersion:      config.ServerConfig.IPVersion,
		connMgr:        NewConnManager(),
		messageHandler: NewMessageHandler(),
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
	addr, err := net.ResolveTCPAddr(s.IPVersion, s.ip+":"+strconv.Itoa(s.port))
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP(s.IPVersion, addr)
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
		if s.connMgr.Len() >= int(config.ServerConfig.MaxConn) {
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
			connection := NewConnection(connID, tcp, s.messageHandler, s.connMgr, s.initializer)
			s.connMgr.Add(connection)
			connection.Start()
		}()
	}
}

var connIDCounter atomic.Int64

func generalConnID() int64 {
	return connIDCounter.Add(1)
}
