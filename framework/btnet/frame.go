package btnet

import (
	"encoding/binary"
	"fmt"
)

/*
┌────────┬──────────┬──────────────┐
│ flags  │ length   │ body         │
│ 1 byte │ 4 byte   │ length byte  │
└────────┴──────────┴──────────────┘
*/

// FrameHeaderSize flags(1) + length(4)
const FrameHeaderSize = 5

// flags 位定义；bit2-7 预留，必须为 0
const (
	FlagCompress uint8 = 1 << 0
	FlagEncrypt  uint8 = 1 << 1

	// 当前支持的 flags （暂无）
	flagsSupported uint8 = 0
)

var frameByteOrder = binary.BigEndian

type Frame struct {
	Flags uint8
	Body  []byte
}

type frameHeader struct {
	Flags   uint8
	BodyLen uint32
}

func decodeFrameHeader(b []byte) frameHeader {
	return frameHeader{Flags: b[0], BodyLen: frameByteOrder.Uint32(b[1:FrameHeaderSize])}
}

// PutFrameHeader 把帧头写到 b[:FrameHeaderSize]，上层可一次分配直接写出整帧
func PutFrameHeader(b []byte, flags uint8, bodyLen int) {
	b[0] = flags
	frameByteOrder.PutUint32(b[1:FrameHeaderSize], uint32(bodyLen))
}

func checkFlags(flags uint8) error {
	//"按位清除"（AND NOT）,即 a & (^b)。
	//==0,flags 里置 1 的位都在支持范围内，校验通过
	if flags&^flagsSupported != 0 {
		return fmt.Errorf("%w: 0x%02x", ErrUnsupportedFlags, flags)
	}
	return nil
}

func frameLen(header frameHeader, maxFrameSize int) (int, error) {
	n := int64(FrameHeaderSize) + int64(header.BodyLen)
	if n > int64(maxFrameSize) {
		return 0, fmt.Errorf("%w: bodyLen=%d, maxFrameSize=%d", ErrFrameTooLarge, header.BodyLen, maxFrameSize)
	}
	return int(n), nil
}

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

func (d *FrameDecoder) Decode(b []byte) ([]*Frame, error) {
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

	var frames []*Frame
	off := 0
	for len(data)-off >= FrameHeaderSize {
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
		copy(body, data[off+FrameHeaderSize:off+totalLen])
		frames = append(frames, &Frame{Flags: header.Flags, Body: body})
		off += totalLen
	}

	// 剩余半包挪到 buffer 头部
	d.buffer = append(d.buffer[:0], data[off:]...)
	return frames, nil
}
