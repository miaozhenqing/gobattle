package btnet

import (
	"fmt"
	"gobattle/btiface"
	utils "gobattle/util"
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
		name:           utils.ServerConfig.Name,
		ip:             utils.ServerConfig.Host,
		port:           utils.ServerConfig.TcpPort,
		IPVersion:      utils.ServerConfig.IPVersion,
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
	go func() {
		addr, err := net.ResolveTCPAddr(utils.ServerConfig.IPVersion, s.ip+":"+strconv.Itoa(s.port))
		if err != nil {
			panic(err)
		}
		listener, err := net.ListenTCP(utils.ServerConfig.IPVersion, addr)
		if err != nil {
			panic(err)
		}

		for {
			tcp, err := listener.AcceptTCP()
			if err != nil {
				fmt.Println(err)
				continue
			}

			go func() {
				connID := generalConnID()
				connection := NewConnection(connID, tcp, s.messageHandler)
				s.connMgr.Add(connection)
				connection.Start()
			}()
		}
	}()
}

func generalConnID() int64 {
	connID := time.Now().UnixMilli()
	return connID
}
