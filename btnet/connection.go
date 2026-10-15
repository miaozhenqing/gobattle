package btnet

import (
	"context"
	"errors"
	"gobattle/btiface"
	"gobattle/btlog"
	"gobattle/config"
	"net"
	"time"
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
	pipeline       btiface.IPipeline

	workerID int
}

func NewConnection(id int64, conn *net.TCPConn, messageHandler btiface.IMessageHandler, connMgr btiface.IConnManager, initializer btiface.PipelineInitializer) *Connection {
	c := &Connection{
		id:             id,
		conn:           conn,
		messageHandler: messageHandler,
		msgChan:        make(chan []byte, config.ServerConfig.SendQueueSize),
		frameDecoder:   NewFrameDecoder(),
		connMgr:        connMgr,

		workerID: messageHandler.AllocateWorkerID(id),
	}
	c.cancelCtx, c.cancelFunc = context.WithCancel(context.Background())
	p := NewPipeline(c)
	initializer(p)
	c.pipeline = p
	return c
}

func (c *Connection) GetID() int64 {
	return c.id
}
func (c *Connection) Start() {

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
	c.pipeline.FireWrite(NewMessage(uint32(msgId), bytes))
}

func (c *Connection) writeRaw(data []byte) {
	select {
	case c.msgChan <- data:
	case <-c.cancelCtx.Done():
	default:
		btlog.Ins().Errorf("send queue full, connID:%v", c.GetID())
		c.Stop()
	}
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
				//这里只往通道放消息，阻塞了起到背压作用
				c.messageHandler.HandleMessage(c, packet)
			}
		}

	}
}
func (c *Connection) startWrite() {
	defer c.Stop()
	for {
		select {
		case data := <-c.msgChan:
			if err := c.write(data); err != nil {
				return
			}
		case <-c.cancelCtx.Done():
			btlog.Ins().Infof("write done, connID:%v", c.GetID())
			return
		}
	}
}

func (c *Connection) write(data []byte) error {
	if timeout := config.ServerConfig.WriteTimeout; timeout > 0 {
		if err := c.conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
			btlog.Ins().Errorf("set write deadline error, connID:%v, err:%v", c.GetID(), err)
			return err
		}
	}
	if _, err := c.conn.Write(data); err != nil {
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			btlog.Ins().Errorf("write timeout, connID:%v", c.GetID())
		} else {
			btlog.Ins().Errorf("write error, connID:%v, error:%v", c.GetID(), err)
		}
		return err
	}
	return nil
}

func (c *Connection) GetWorkerID() int {
	return c.workerID
}

func (c *Connection) GetPipeline() btiface.IPipeline {
	return c.pipeline
}
