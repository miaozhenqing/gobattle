package btnet

import (
	"errors"
	"fmt"
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

func (d *FrameDecoder) Decode(b []byte) ([][]byte, error) {
	if len(b) == 0 {
		return nil, nil
	}

	d.buffer = append(d.buffer, b...)

	var packets [][]byte

	for {
		// Header都不够
		if len(d.buffer) < headerSize {
			break
		}

		header, err := decodeFrameHeader(d.buffer)
		if err != nil {
			return nil, err
		}
		dataLen := header.DataLen

		// 校验长度
		if uint64(headerSize)+uint64(dataLen) > uint64(d.maxFrameSize) {
			return nil, fmt.Errorf(
				"%w: dataLen=%d, maxFrameSize=%d",
				ErrFrameTooLarge,
				dataLen,
				d.maxFrameSize,
			)
		}

		// 完整帧长度
		totalLen := headerSize + int(dataLen)

		// data 还没收完整
		if len(d.buffer) < totalLen {
			break
		}

		// 复制完整帧，避免和内部 buffer 共享底层数组
		packet := make([]byte, totalLen)
		copy(packet, d.buffer[:totalLen])

		packets = append(packets, packet)

		// 移除已经解析完成的
		d.buffer = d.buffer[totalLen:]

		// 没有剩余数据
		if len(d.buffer) == 0 {
			break
		}
	}

	return packets, nil
}
