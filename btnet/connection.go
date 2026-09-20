package btnet

import (
	"fmt"
	"net"
)

type Connection struct {
	ID           int64
	conn         *net.TCPConn
	ExitBuffChan chan bool
	isClosed     bool
}

func NewConnection(id int64, conn *net.TCPConn) *Connection {
	return &Connection{
		ID:           id,
		conn:         conn,
		ExitBuffChan: make(chan bool),
	}
}

func (c *Connection) GetID() int64 {
	return c.ID
}
func (c *Connection) Start() {
	go c.StartRead()
	go c.StartWrite()
	<-c.ExitBuffChan
}
func (c *Connection) Stop() {
	if c.isClosed {
		return
	}
	c.conn.Close()
	c.isClosed = true
	c.ExitBuffChan <- true
	close(c.ExitBuffChan)
}
func (c *Connection) StartRead() {
	defer c.Stop()
	for {
		if c.isClosed {
			return
		}
		//创建数组
		buf := make([]byte, 1024)
		//读取数据到数组
		_, err := c.conn.Read(buf)
		if err != nil {
			fmt.Println(err)
			continue
		}
		//解码
	}
}
func (c *Connection) StartWrite() {

}
