package main

import (
	"fmt"
	"gobattle/btnet"
	"gobattle/config"
	"gobattle/router"
	utils "gobattle/util"
	"net"
	"strconv"
	"time"
)

func main() {
	s := btnet.NewServer()

	s.SetSerializer(btnet.NewSerializer())
	s.AddRouter(1001, router.NewPingRouter())

	s.Start()
	time.Sleep(3)
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
	for {
		s := "hello world"
		message := btnet.NewMessage(uint32(utils.MessageIDLogin), []byte(s))

		buf := btnet.NewSerializer().Serialize(message)

		conn.Write(buf)

		// 接收返回值
		response := make([]byte, 1024)

		n, err := conn.Read(response)
		if err != nil {
			fmt.Println("read error:", err)
			return
		}
		fmt.Println("receive:", string(response[:n]))

		time.Sleep(time.Second * 3)
	}

}
