package btiface

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
