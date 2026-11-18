package btrpc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"gobattle/framework/btnet"
)

type Packet struct {
	Type    btnet.MessageType //1
	Seq     uint32            //4
	Code    uint32            //4
	MsgId   uint32            //4
	UserId  int64             //8
	Method  string            //2
	Payload []byte
}

var ErrInvalidPacket = errors.New("invalid packet")

const packetFixedSize = 1 + 4 + 4 + 4 + 8 + 2

var packetByteOrder = binary.BigEndian

func encodePacket(p *Packet) []byte {
	bodyLen := packetFixedSize + len(p.Method) + len(p.Payload)
	buf := make([]byte, btnet.FrameHeaderSize+bodyLen)
	//设置帧头
	btnet.PutFrameHeader(buf, 0, bodyLen)
	b := buf[btnet.FrameHeaderSize:]
	b[0] = byte(p.Type)
	packetByteOrder.PutUint32(b[1:5], p.Seq)
	packetByteOrder.PutUint32(b[5:9], p.Code)
	packetByteOrder.PutUint32(b[9:13], p.MsgId)
	packetByteOrder.PutUint64(b[13:21], uint64(p.UserId))
	packetByteOrder.PutUint16(b[21:23], uint16(len(p.Method)))
	start := 23
	//实际写入的字节数 i
	i := copy(b[start:], p.Method)
	copy(b[start+i:], p.Payload)
	return buf
}

func decodePacket(body []byte) (*Packet, error) {
	if len(body) < packetFixedSize {
		return nil, fmt.Errorf("%w: len:%v", ErrInvalidPacket, len(body))
	}
	messageType := btnet.MessageType(body[0])
	if messageType < btnet.MessageRequest || messageType > btnet.MessagePush {
		return nil, fmt.Errorf("%w: typ:%v", ErrInvalidPacket, messageType)
	}
	methodLen := packetByteOrder.Uint16(body[21:23])
	if len(body) < int(packetFixedSize+methodLen) {
		return nil, fmt.Errorf("%w: typ:%v", ErrInvalidPacket, messageType)
	}
	off := 23 + methodLen
	return &Packet{
		Type:    messageType,
		Seq:     packetByteOrder.Uint32(body[1:5]),
		Code:    packetByteOrder.Uint32(body[5:9]),
		MsgId:   packetByteOrder.Uint32(body[9:13]),
		UserId:  int64(packetByteOrder.Uint64(body[13:21])),
		Method:  string(body[23:off]),
		Payload: body[off:],
	}, nil
}

var (
	_ btnet.InHandler  = PacketCodec{}
	_ btnet.OutHandler = PacketCodec{}
)

type PacketCodec struct{}

func (PacketCodec) OnRead(ctx *btnet.HandlerContext, msg any) {
	f, ok := msg.(*btnet.Frame)
	if !ok {
		ctx.FireRead(msg)
		return
	}
	p, err := decodePacket(f.Body)
	if err != nil {
		ctx.FireException(err)
		return
	}
	ctx.FireRead(p)
}

func (PacketCodec) OnWrite(ctx *btnet.HandlerContext, msg any) {
	p, ok := msg.(*Packet)
	if !ok {
		ctx.FireWrite(msg)
		return
	}
	ctx.FireWrite(encodePacket(p))
}
