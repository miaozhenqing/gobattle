package btnet

import (
	"errors"
	"fmt"
	"gobattle/btiface"
)

/*
┌────────┬──────────┬──────────────┐
│ msgId  │ dataLen  │ data         │
│ 4 byte │ 4 byte   │ dataLen byte │
└────────┴──────────┴──────────────┘
*/

var ErrFrameTooLarge = errors.New("frame too large")

type FrameDecoder struct {
	maxFrameSize uint32
	buffer       []byte
}

func NewFrameDecoder(maxFrameSize, initBufSize uint32) *FrameDecoder {
	return &FrameDecoder{
		maxFrameSize: maxFrameSize,
		buffer:       make([]byte, 0, initBufSize),
	}
}

func (d *FrameDecoder) Decode(b []byte) ([]btiface.IMessage, error) {
	if len(b) == 0 {
		return nil, nil
	}

	data := b
	// 缓存里有残留，说明上一次解析时半包，需要合并当前数据；
	// 如果没有缓存，直接在b上解析，不用复制到buffer
	if len(d.buffer) > 0 {
		d.buffer = append(d.buffer, b...)
		data = d.buffer
	}

	var msgs []btiface.IMessage
	off := 0
	for len(data)-off >= headerSize {
		header, err := decodeFrameHeader(data[off:])
		if err != nil {
			return nil, err
		}
		if uint64(headerSize)+uint64(header.DataLen) > uint64(d.maxFrameSize) {
			return nil, fmt.Errorf("%w: dataLen=%d, maxFrameSize=%d", ErrFrameTooLarge, header.DataLen, d.maxFrameSize)
		}
		totalLen := headerSize + int(header.DataLen)
		if len(data)-off < totalLen {
			break
		}

		payload := make([]byte, header.DataLen)
		copy(payload, data[off+headerSize:off+totalLen])
		msgs = append(msgs, NewMessage(header.MessageID, payload))
		off += totalLen
	}

	// 剩余半包挪到 buffer 头部
	d.buffer = append(d.buffer[:0], data[off:]...)
	return msgs, nil
}
