package btnet

import (
	"fmt"
	"gobattle/btiface"
	utils "gobattle/util"
	"net"
)

type Connection struct {
	id             int64
	conn           *net.TCPConn
	exitBuffChan   chan bool
	isClosed       bool
	messageHandler btiface.IMessageHandler
	msgChan        chan []byte
	frameDecoder   btiface.IFrameDecoder
}

func NewConnection(id int64, conn *net.TCPConn, messageHandler btiface.IMessageHandler) *Connection {
	return &Connection{
		id:             id,
		conn:           conn,
		exitBuffChan:   make(chan bool),
		messageHandler: messageHandler,
		msgChan:        make(chan []byte),
		frameDecoder:   NewFrameDecoder(),
	}
}

func (c *Connection) GetID() int64 {
	return c.id
}
func (c *Connection) Start() {
	go c.startRead()
	go c.startWrite()
	<-c.exitBuffChan
}
func (c *Connection) Stop() {
	if c.isClosed {
		return
	}
	c.conn.Close()
	c.isClosed = true
	c.exitBuffChan <- true
	close(c.exitBuffChan)
}

func (c *Connection) SendMsg(msgId int32, bytes []byte) {
	if c.isClosed {
		return
	}
	c.msgChan <- bytes
}
func (c *Connection) startRead() {
	defer c.Stop()
	buf := make([]byte, utils.ServerConfig.IOReadBuffSize)

	for {
		if c.isClosed {
			return
		}
		//读取数据到数组
		size, err := c.conn.Read(buf)
		if err != nil {
			fmt.Println("read error:", err)
			return
		}
		if size == 0 {
			continue
		}
		//解包
		packets, err := c.frameDecoder.Decode(buf[:size])
		if err != nil {
			fmt.Println("decode error:", err)
			return
		}
		if len(packets) == 0 {
			fmt.Println("decode no packet")
		}
		//解码
		for _, packet := range packets {
			req := NewRequest(c, packet)
			go c.messageHandler.HandleMessage(req)
		}
	}
}
func (c *Connection) startWrite() {
	for {
		select {
		case data := <-c.msgChan:
			c.conn.Write(data)
		case <-c.exitBuffChan:
			return
		}
	}
}
