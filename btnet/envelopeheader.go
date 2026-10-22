package btnet

import (
	"errors"
	"fmt"
	"gobattle/btiface"
)

/*
┌────────┬────────┬────────┬────────┬───────────┐
│ type   │ seq    │ msgId  │ code   │ payload   │
│ 1 byte │ 4 byte │ 4 byte │ 4 byte │ 剩余字节   │
└────────┴────────┴────────┴────────┴───────────┘
*/
const envelopeHeaderSize = 13

var ErrInvalidEnvelope = errors.New("invalid envelope")

func encodeEnvelope(env *btiface.Envelope) []byte {
	buf := make([]byte, envelopeHeaderSize+len(env.Payload))
	buf[0] = byte(env.Type)
	frameByteOrder.PutUint32(buf[1:5], env.Seq)
	frameByteOrder.PutUint32(buf[5:9], env.MsgId)
	frameByteOrder.PutUint32(buf[9:13], env.Code)
	copy(buf[envelopeHeaderSize:], env.Payload)
	return buf
}

func decodeEnvelope(body []byte) (*btiface.Envelope, error) {
	if len(body) < envelopeHeaderSize {
		return nil, fmt.Errorf("%w: len:%v", ErrInvalidEnvelope, len(body))
	}
	typ := btiface.MsgType(body[0])
	if typ < btiface.MsgTypeRequest || typ > btiface.MsgTypePush {
		return nil, fmt.Errorf("%w: typ:%v", ErrInvalidEnvelope, typ)
	}
	return &btiface.Envelope{
		Type:    typ,
		Seq:     frameByteOrder.Uint32(body[1:5]),
		MsgId:   frameByteOrder.Uint32(body[5:9]),
		Code:    frameByteOrder.Uint32(body[9:13]),
		Payload: body[envelopeHeaderSize:],
	}, nil
}

func EncodeEnvelope(env *btiface.Envelope) []byte {
	return encodeFrame(0, encodeEnvelope(env))
}

func DecodeEnvelope(f btiface.IFrame) (*btiface.Envelope, error) {
	return decodeEnvelope(f.GetBody())
}
