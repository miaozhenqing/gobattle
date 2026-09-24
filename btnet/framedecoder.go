package btnet

import (
	"encoding/binary"
	"errors"
	"fmt"

	utils "gobattle/util"
)

/*
┌────────┬──────────┬──────────────┐
│ msgId  │ dataLen  │ data         │
│ 4 byte │ 4 byte   │ dataLen byte │
└────────┴──────────┴──────────────┘
*/

const HeaderSize = 8

var (
	MaxFrameSize     = utils.ServerConfig.MaxPacketSize
	ErrFrameTooLarge = errors.New("frame too large")
)

type FrameDecoder struct {
	buffer []byte
}

func NewFrameDecoder() *FrameDecoder {
	return &FrameDecoder{buffer: make([]byte, 0, utils.ServerConfig.IOReadBuffSize)}
}

func (d *FrameDecoder) Decode(b []byte) ([][]byte, error) {
	if len(b) == 0 {
		return nil, nil
	}

	d.buffer = append(d.buffer, b...)

	var packets [][]byte

	for {
		// Header都不够
		if len(d.buffer) < HeaderSize {
			break
		}

		dataLen := binary.BigEndian.Uint32(d.buffer[4:8])

		// 校验长度
		if uint64(HeaderSize)+uint64(dataLen) > uint64(MaxFrameSize) {
			return nil, fmt.Errorf(
				"%w: dataLen=%d, maxFrameSize=%d",
				ErrFrameTooLarge,
				dataLen,
				MaxFrameSize,
			)
		}

		// 完整帧长度
		totalLen := HeaderSize + int(dataLen)

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
