package btnet

import (
	"errors"
	"gobattle/btiface"
)

var ErrFrameTooLarge = errors.New("frame too large")

type FrameDecoder struct {
	maxFrameSize int
	buffer       []byte
}

func NewFrameDecoder(maxFrameSize, initBufSize int) *FrameDecoder {
	return &FrameDecoder{
		maxFrameSize: maxFrameSize,
		buffer:       make([]byte, 0, initBufSize),
	}
}

func (d *FrameDecoder) Decode(b []byte) ([]btiface.IFrame, error) {
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

	var frames []btiface.IFrame
	off := 0
	for len(data)-off >= headerSize {
		header := decodeFrameHeader(data[off:])
		if err := checkFlags(header.Flags); err != nil {
			return nil, err
		}
		totalLen, err := frameLen(header, d.maxFrameSize)
		if err != nil {
			return nil, err
		}
		if len(data)-off < totalLen {
			break
		}
		body := make([]byte, header.BodyLen)
		copy(body, data[off+headerSize:off+totalLen])
		frames = append(frames, NewFrame(header.Flags, body))
		off += totalLen
	}

	// 剩余半包挪到 buffer 头部
	d.buffer = append(d.buffer[:0], data[off:]...)
	return frames, nil
}
