package btnet

import (
	"encoding/binary"
	"gobattle/btiface"
)

type Serializer struct {
}

func NewSerializer() *Serializer {
	return &Serializer{}
}

func (s Serializer) Serialize(msg btiface.IMessage) []byte {
	b := make([]byte, 8+msg.GetDataLen())
	binary.BigEndian.PutUint32(b[0:4], msg.GetMsgId())
	binary.BigEndian.PutUint32(b[4:8], msg.GetDataLen())
	copy(b[8:], msg.GetData())
	return b
}

func (s Serializer) Deserialize(bytes []byte) (btiface.IMessage, error) {
	msgId := binary.BigEndian.Uint32(bytes[0:4])
	dataLen := binary.BigEndian.Uint32(bytes[4:8])
	payload := make([]byte, dataLen)
	copy(payload, bytes[8:])

	return &Message{
		msgId: msgId,
		data:  payload,
	}, nil
}
