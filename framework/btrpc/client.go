package btrpc

import (
	"context"
	"gobattle/framework/btlog"
	"gobattle/framework/btnet"
	"gobattle/framework/btrouter"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const clientReadBufSize = 64 * 1024

type Client struct {
	node    Node
	conn    net.Conn
	decoder *btnet.FrameDecoder
	codec   btrouter.Codec
	onPush  func(p *Packet)

	seq     atomic.Uint32
	writeMu sync.Mutex

	mu      sync.Mutex
	pending map[uint32]chan *Packet

	closeOnce sync.Once
	done      chan struct{}
}

func dialNodeClient(node Node, maxFrameSize int, codec btrouter.Codec, onPush func(*Packet)) (*Client, error) {
	conn, err := net.DialTimeout("tcp", node.Addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	c := &Client{
		node:    node,
		conn:    conn,
		codec:   codec,
		onPush:  onPush,
		decoder: btnet.NewFrameDecoder(maxFrameSize, clientReadBufSize),
		pending: make(map[uint32]chan *Packet),
		done:    make(chan struct{}),
	}
	go c.readPackets()
	return c, nil
}

func (c *Client) Invoke(ctx context.Context, method string, payload []byte) ([]byte, error) {
	return c.sendRequestAndWait(ctx, method, payload)
}

func (c *Client) Codec() btrouter.Codec { return c.codec }

func (c *Client) sendRequestAndWait(ctx context.Context, method string, payload []byte) ([]byte, error) {
	seq := c.nextRequestSeq()
	ch := make(chan *Packet, 1)
	c.mu.Lock()
	c.pending[seq] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, seq)
		c.mu.Unlock()
	}()

	m := GetRequestMeta(ctx)
	p := &Packet{
		Type:    btnet.MessageRequest,
		Seq:     seq,
		Method:  method,
		UserId:  m.UserId,
		Payload: payload,
	}
	//向对端写数据
	if err := c.writePacket(p); err != nil {
		return nil, btrouter.CodeUnavailable
	}

	select {
	//阻塞等待对端回包（readPackets）
	case resp := <-ch:
		if code := btrouter.Code(resp.Code); code != btrouter.CodeOK {
			return nil, code
		}
		return resp.Payload, nil
	case <-ctx.Done():
		return nil, btrouter.CodeDeadlineExceeded
	case <-c.done:
		return nil, btrouter.CodeUnavailable
	}
}

func (c *Client) nextRequestSeq() uint32 {
	for {
		if add := c.seq.Add(1); add != 0 {
			return add
		}
	}
}

func (c *Client) writePacket(p *Packet) error {
	select {
	case <-c.Done():
		return btrouter.CodeUnavailable
	default:
	}
	data := encodePacket(p)
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err := c.conn.Write(data)
	return err
}

func (c *Client) readPackets() {
	defer c.Close()
	buf := make([]byte, clientReadBufSize)
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			btlog.Ins().Error("rpcClientReadError", "node", c.node.NodeId, "addr", c.node.Addr, "err", err)
			return
		}
		frames, err := c.decoder.Decode(buf[:n])
		if err != nil {
			btlog.Ins().Error("rpcClientDecodeError", "node", c.node.NodeId, "addr", c.node.Addr, "err", err)
			return
		}
		for _, frame := range frames {
			p, err := decodePacket(frame.Body)
			if err != nil {
				btlog.Ins().Error("rpcClientPacketDecodeError", "node", c.node.NodeId, "addr", c.node.Addr, "err", err)
				return
			}
			c.handleResponseOrPush(p)
		}
	}
}

func (c *Client) handleResponseOrPush(p *Packet) {
	switch p.Type {
	case btnet.MessageResponse:
		c.mu.Lock()
		ch := c.pending[p.Seq]
		c.mu.Unlock()
		if ch != nil {
			ch <- p
		}
	case btnet.MessagePush:
		if c.onPush != nil {
			c.onPush(p)
		}
	}
}

func (c *Client) Close() error {
	err := c.conn.Close()
	c.markClosed()
	return err
}

func (c *Client) markClosed() {
	c.closeOnce.Do(func() {
		close(c.done)
	})
}

func (c *Client) Done() <-chan struct{} {
	return c.done
}
