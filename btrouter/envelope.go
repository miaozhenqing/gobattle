package btrouter

import (
	"encoding/binary"
	"fmt"
	"gobattle/btnet"
)

type MsgType uint8

const (
	MsgTypeRequest  MsgType = 1
	MsgTypeResponse MsgType = 2
	MsgTypePush     MsgType = 3
)

type Envelope struct {
	Type    MsgType
	Seq     uint32
	MsgId   uint32
	Code    uint32
	Payload []byte
}

/*
┌────────┬────────┬────────┬────────┬───────────┐
│ type   │ seq    │ msgId  │ code   │ payload   │
│ 1 byte │ 4 byte │ 4 byte │ 4 byte │ 剩余字节   │
└────────┴────────┴────────┴────────┴───────────┘
*/
const envelopeHeaderSize = 13

var envelopeByteOrder = binary.BigEndian

func encodeEnvelope(env *Envelope) []byte {
	bodyLen := envelopeHeaderSize + len(env.Payload)
	buf := make([]byte, btnet.FrameHeaderSize+bodyLen)
	btnet.PutFrameHeader(buf, 0, bodyLen)
	b := buf[btnet.FrameHeaderSize:]
	b[0] = byte(env.Type)
	envelopeByteOrder.PutUint32(b[1:5], env.Seq)
	envelopeByteOrder.PutUint32(b[5:9], env.MsgId)
	envelopeByteOrder.PutUint32(b[9:13], env.Code)
	copy(b[envelopeHeaderSize:], env.Payload)
	return buf
}

func decodeEnvelope(body []byte) (*Envelope, error) {
	if len(body) < envelopeHeaderSize {
		return nil, fmt.Errorf("%w: len:%v", ErrInvalidEnvelope, len(body))
	}
	typ := MsgType(body[0])
	if typ < MsgTypeRequest || typ > MsgTypePush {
		return nil, fmt.Errorf("%w: typ:%v", ErrInvalidEnvelope, typ)
	}
	return &Envelope{
		Type:    typ,
		Seq:     envelopeByteOrder.Uint32(body[1:5]),
		MsgId:   envelopeByteOrder.Uint32(body[5:9]),
		Code:    envelopeByteOrder.Uint32(body[9:13]),
		Payload: body[envelopeHeaderSize:],
	}, nil
}

var (
	_ btnet.InHandler  = (*EnvelopeCodec)(nil)
	_ btnet.OutHandler = (*EnvelopeCodec)(nil)
)

type EnvelopeCodec struct {
}

func NewEnvelopeCodec() *EnvelopeCodec {
	return &EnvelopeCodec{}
}

func (e *EnvelopeCodec) OnRead(ctx *btnet.HandlerContext, msg any) {
	f, ok := msg.(*btnet.Frame)
	if !ok {
		ctx.FireRead(msg)
		return
	}
	env, err := decodeEnvelope(f.Body)
	if err != nil {
		ctx.FireException(err)
		return
	}
	ctx.FireRead(env)
}

func (e *EnvelopeCodec) OnWrite(ctx *btnet.HandlerContext, msg any) {
	env, ok := msg.(*Envelope)
	if !ok {
		ctx.FireWrite(msg)
		return
	}
	ctx.FireWrite(encodeEnvelope(env))
}
