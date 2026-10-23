package btrouter

import (
	"context"
	"gobattle/btnet"
	"net"
	"sync"
	"sync/atomic"
)

const clientReadBufSize = 4096

type Client struct {
	conn    net.Conn
	codec   Codec
	decoder *btnet.FrameDecoder
	onPush  func(msgId uint32, data []byte)

	seq     atomic.Uint32
	writeMu sync.Mutex

	mu      sync.Mutex
	pending map[uint32]chan *Envelope

	closeOnce sync.Once
	done      chan struct{}
}

func Dial(addr string, maxFrameSize int, codec Codec, onPush func(msgId uint32, data []byte)) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	if codec == nil {
		codec = JSONCodec{}
	}
	c := &Client{
		conn:    conn,
		codec:   codec,
		decoder: btnet.NewFrameDecoder(maxFrameSize, clientReadBufSize),
		onPush:  onPush,
		pending: make(map[uint32]chan *Envelope),
		done:    make(chan struct{}),
	}
	go c.readLoop()
	return c, nil
}

func Call[Resp any](ctx context.Context, c *Client, msgId uint32, req any) (*Resp, error) {
	data, err := c.roundTrip(ctx, msgId, req)
	if err != nil {
		return nil, err
	}
	resp := new(Resp)
	if len(data) > 0 {
		if err := c.codec.Unmarshal(data, resp); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func (c *Client) Notify(msgId uint32, req any) error {
	payload, err := c.marshal(req)
	if err != nil {
		return err
	}
	return c.write(&Envelope{Type: MsgTypeRequest, MsgId: msgId, Payload: payload})
}

func (c *Client) Close() error {
	err := c.conn.Close()
	c.shutdown()
	return err
}

func (c *Client) roundTrip(ctx context.Context, msgId uint32, req any) ([]byte, error) {
	payload, err := c.marshal(req)
	if err != nil {
		return nil, err
	}

	seq := c.nextSeq()
	ch := make(chan *Envelope, 1)
	c.mu.Lock()
	c.pending[seq] = ch
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.pending, seq)
		c.mu.Unlock()
	}()

	if err := c.write(&Envelope{Type: MsgTypeRequest, Seq: seq, MsgId: msgId, Payload: payload}); err != nil {
		return nil, err
	}

	select {
	case env := <-ch:
		if env.Code != CodeOK {
			return nil, &Error{Code: env.Code, Msg: string(env.Payload)}
		}
		return env.Payload, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.done:
		return nil, ErrClientClosed
	}
}

func (c *Client) nextSeq() uint32 {
	for {
		if seq := c.seq.Add(1); seq != 0 {
			return seq
		}
	}
}

func (c *Client) marshal(req any) ([]byte, error) {
	if req == nil {
		return nil, nil
	}
	return c.codec.Marshal(req)
}

func (c *Client) write(env *Envelope) error {
	select {
	case <-c.done:
		return ErrClientClosed
	default:
	}
	data := encodeEnvelope(env)
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err := c.conn.Write(data)
	return err
}

func (c *Client) readLoop() {
	defer c.Close()
	buf := make([]byte, clientReadBufSize)
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			return
		}
		frames, err := c.decoder.Decode(buf[:n])
		if err != nil {
			return
		}
		for _, f := range frames {
			env, err := decodeEnvelope(f.Body)
			if err != nil {
				return
			}
			c.deliver(env)
		}
	}
}

func (c *Client) deliver(env *Envelope) {
	switch env.Type {
	case MsgTypeResponse:
		c.mu.Lock()
		ch := c.pending[env.Seq]
		c.mu.Unlock()
		if ch != nil {
			ch <- env
		}
	case MsgTypePush:
		if c.onPush != nil {
			c.onPush(env.MsgId, env.Payload)
		}
	}
}

func (c *Client) shutdown() {
	c.closeOnce.Do(func() {
		close(c.done)
	})
}
