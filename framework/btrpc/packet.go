package btrpc

import (
	"encoding/binary"
	"errors"
	"gobattle/framework/btnet"
)

type Packet struct {
	Type      btnet.MessageType //1
	Seq       uint32            //4
	Code      uint32            //4
	Method    string            //
	UserId    int64             //8
	GatewayId int64             //8
	Payload   []byte
}

var ErrInvalidPacket = errors.New("invalid packet")

const packetFixedSize = 1 + 4 + 4 + 2 + 8 + 8

var packetByteOrder = binary.BigEndian

func encodePacket(p *Packet) []byte {
	//todo
	return nil
}

func decodePacket(body []byte) (*Packet, error) {
	return nil, errors.New("TODO: decode RPC packet")
}

var (
	_ btnet.InHandler  = PacketCodec{}
	_ btnet.OutHandler = PacketCodec{}
)

type PacketCodec struct{}

func (PacketCodec) OnRead(ctx *btnet.HandlerContext, msg any) {
	panic("TODO: decode RPC frame")
}

func (PacketCodec) OnWrite(ctx *btnet.HandlerContext, msg any) {
	panic("TODO: encode RPC frame")
}
