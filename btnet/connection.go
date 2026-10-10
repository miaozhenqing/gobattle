package btnet

import (
	"context"
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/config"
	"net"
)

type Connection struct {
	id             int64
	conn           *net.TCPConn
	messageHandler btiface.IMessageHandler
	msgChan        chan []byte
	frameDecoder   btiface.IFrameDecoder
	connMgr        btiface.IConnManager
	cancelCtx      context.Context
	cancelFunc     context.CancelFunc

	workerID int
}

func NewConnection(id int64, conn *net.TCPConn, messageHandler btiface.IMessageHandler, connMgr btiface.IConnManager) *Connection {
	return &Connection{
		id:             id,
		conn:           conn,
		messageHandler: messageHandler,
		msgChan:        make(chan []byte),
		frameDecoder:   NewFrameDecoder(),
		connMgr:        connMgr,

		workerID: messageHandler.AllocateWorkerID(id),
	}
}

func (c *Connection) GetID() int64 {
	return c.id
}
func (c *Connection) Start() {
	c.cancelCtx, c.cancelFunc = context.WithCancel(context.Background())
	go c.startRead()
	go c.startWrite()

	select {
	case <-c.cancelCtx.Done():
		c.destroy()
		return
	}
}
func (c *Connection) Stop() {
	c.cancelFunc()
}

func (c *Connection) destroy() {
	c.conn.Close()
	c.connMgr.Remove(c.GetID())
}

func (c *Connection) SendMsg(msgId int32, bytes []byte) {
	c.msgChan <- bytes
}
func (c *Connection) startRead() {
	defer c.Stop()
	buf := make([]byte, config.ServerConfig.IOReadBuffSize)

	for {
		select {
		case <-c.cancelCtx.Done():
			btlog.Ins().Info("read done")
			return
		default:
			//读取数据到数组
			size, err := c.conn.Read(buf)
			if err != nil {
				btlog.Ins().Errorf("read error:%v", err)
				return
			}
			if size == 0 {
				continue
			}
			//解包
			packets, err := c.frameDecoder.Decode(buf[:size])
			if err != nil {
				btlog.Ins().Errorf("decode error:%v", err)
				return
			}
			if len(packets) == 0 {
				btlog.Ins().Warn("decode no packet")
			}
			//解码
			for _, packet := range packets {
				req := GetRequest(c, packet)
				go c.messageHandler.HandleMessage(req)
			}
		}

	}
}
func (c *Connection) startWrite() {
	for {
		select {
		case data := <-c.msgChan:
			c.conn.Write(data)
		case <-c.cancelCtx.Done():
			btlog.Ins().Info("write done")
			return
		}
	}
}

func (c *Connection) GetWorkerID() int {
	return c.workerID
}
