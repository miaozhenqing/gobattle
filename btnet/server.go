package btnet

import (
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/config"
	"net"
	"strconv"
	"time"
)

type server struct {
	name           string
	ip             string
	port           int
	IPVersion      string
	connMgr        btiface.IConnManager
	messageHandler btiface.IMessageHandler
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

func (s *server) SetSerializer(ser btiface.ISerializer) {
	s.messageHandler.SetSerializer(ser)
}
func (s *server) AddRouter(msgId uint32, r btiface.IRouter) {
	s.messageHandler.AddRouter(msgId, r)
}

func (s *server) Start() {
	addr, err := net.ResolveTCPAddr(config.ServerConfig.IPVersion, s.ip+":"+strconv.Itoa(s.port))
	if err != nil {
		panic(err)
	}
	go func() {

		listener, err := net.ListenTCP(config.ServerConfig.IPVersion, addr)
		if err != nil {
			panic(err)
		}

		for {
			if s.connMgr.Len() > int(config.ServerConfig.MaxConn) {
				//todo 先暴力sleep
				time.Sleep(1000 * time.Millisecond)
				btlog.Ins().Warn("over max conn")
				continue
			}

			tcp, err := listener.AcceptTCP()
			if err != nil {
				btlog.Ins().Errorf("accept tcp error:%v", err)
				continue
			}

			go func() {
				connID := generalConnID()
				connection := NewConnection(connID, tcp, s.messageHandler, s.connMgr)
				s.connMgr.Add(connection)
				connection.Start()
			}()
		}
	}()
}

func generalConnID() int64 {
	//todo 暂时使用时间戳
	connID := time.Now().UnixNano()
	return connID
}
