package main

import (
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/btnet"
	"gobattle/config"
	"gobattle/handler"
	"gobattle/router"
	utils "gobattle/util"
	"net"
	"strconv"
	"time"
)

func main() {
	s := btnet.NewServer()

	serializer := btnet.NewSerializer()

	dispatcher := handler.NewDispatchHandler()
	dispatcher.AddRouter(1001, router.NewPingRouter())

	s.SetPipelineInitializer(func(p btiface.IPipeline) {
		p.AddLast(handler.NewDecodeHandler(serializer))
		p.AddLast(dispatcher)
		p.AddLast(handler.NewEncodeHandler(serializer))
	})

	err := s.Start()
	if err != nil {
		panic("server start error")
	}
	client := NewClient()
	client.Test()
}

type Client struct {
}

func NewClient() *Client {
	return &Client{}
}
func (c *Client) Test() {
	resolveTCPAddr, _ := net.ResolveTCPAddr("tcp", config.ServerConfig.Host+":"+strconv.Itoa(config.ServerConfig.TcpPort))

	conn, err := net.DialTCP("tcp", nil, resolveTCPAddr)
	if err != nil {
		panic(err)
	}
	serializer := btnet.NewSerializer()
	for {
		s := "hello world"
		message := btnet.NewMessage(uint32(utils.MessageIDLogin), []byte(s))

		buf := serializer.Serialize(message)

		conn.Write(buf)

		// 接收返回值
		response := make([]byte, 1024)

		n, err := conn.Read(response)
		if err != nil {
			btlog.Ins().Errorf("read error:%v", err)
			return
		}
		deserialize, err := serializer.Deserialize(response[:n])
		btlog.Ins().Info("receive:", string(deserialize.GetData()))

		time.Sleep(time.Second * 3)
	}

}
