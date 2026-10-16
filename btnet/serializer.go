package btnet

import (
	"fmt"
	"gobattle/btiface"
)

type Serializer struct {
	maxFrameSize int
}

func NewSerializer(maxFrameSize int) *Serializer {
	return &Serializer{maxFrameSize: maxFrameSize}
}

func (s *Serializer) Serialize(msg btiface.IMessage) []byte {
	header, err := encodeFrameHeader(frameHeader{
		MessageID: msg.GetMsgId(),
		DataLen:   msg.GetDataLen(),
	})
	if err != nil {
		return nil
	}
	return append(header, msg.GetData()...)
}

func (s *Serializer) Deserialize(data []byte) (btiface.IMessage, error) {
	header, err := decodeFrameHeader(data)
	if err != nil {
		return nil, err
	}
	totalLen, err := frameLen(header, s.maxFrameSize)
	if err != nil {
		return nil, err
	}
	if len(data) != totalLen {
		return nil, fmt.Errorf("frame length mismatch: declared=%d, actual=%d", header.DataLen, len(data)-headerSize)
	}

	payload := append([]byte(nil), data[headerSize:]...)
	return &Message{msgId: header.MessageID, data: payload}, nil
}
