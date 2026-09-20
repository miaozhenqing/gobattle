package btnet

import "net"

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

	}
}
func (c *Connection) StartWrite() {

}
