package btnet

import (
	"fmt"
	"gobattle/btiface"
	"net"
	"strconv"
	"time"
)

type server struct {
	Name    string
	IP      string
	Port    int
	ConnMgr btiface.IConnManager
}

func NewServer(IP string, port int) *server {
	s := &server{
		Name:    "battle",
		IP:      IP,
		Port:    port,
		ConnMgr: NewConnManager(),
	}
	return s
}
func (s *server) Start() {
	go func() {
		var IPVersion = "tcp4"
		addr, err := net.ResolveTCPAddr(IPVersion, s.IP+":"+strconv.Itoa(s.Port))
		if err != nil {
			panic(err)
		}
		listener, err := net.ListenTCP(IPVersion, addr)
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
				connID := time.Now().UnixMilli()
				connection := NewConnection(connID, tcp)
				s.ConnMgr.Add(connection)
				connection.Start()
			}()
		}
	}()
}
