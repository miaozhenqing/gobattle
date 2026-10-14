package btnet

import (
	"fmt"
	"gobattle/btiface"
)

type Serializer struct {
}

func NewSerializer() *Serializer {
	return &Serializer{}
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
	if uint64(headerSize)+uint64(header.DataLen) > uint64(MaxFrameSize) {
		return nil, fmt.Errorf("%w: dataLen=%d", ErrFrameTooLarge, header.DataLen)
	}
	if uint64(len(data)) != uint64(headerSize)+uint64(header.DataLen) {
		return nil, fmt.Errorf("frame length mismatch: declared=%d, actual=%d", header.DataLen, len(data)-headerSize)
	}

	payload := append([]byte(nil), data[headerSize:]...)
	return &Message{msgId: header.MessageID, data: payload}, nil
}
