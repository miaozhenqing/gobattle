package btnet

import (
	"encoding/binary"
	"errors"
	"fmt"
)

/*
┌────────┬──────────┬──────────────┐
│ flags  │ length   │ body         │
│ 1 byte │ 4 byte   │ length byte  │
└────────┴──────────┴──────────────┘
*/

// flags(1) + length(4)
const headerSize = 5

// flags 位定义；bit2-7 预留，必须为 0
const (
	FlagCompress uint8 = 1 << 0
	FlagEncrypt  uint8 = 1 << 1

	// 当前支持的 flags （暂无）
	flagsSupported uint8 = 0
)

var frameByteOrder = binary.BigEndian

var ErrUnsupportedFlags = errors.New("unsupported frame flags")

type frameHeader struct {
	Flags   uint8
	BodyLen uint32
}

func decodeFrameHeader(b []byte) frameHeader {
	return frameHeader{Flags: b[0], BodyLen: frameByteOrder.Uint32(b[1:headerSize])}
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
	n := int64(headerSize) + int64(header.BodyLen)
	if n > int64(maxFrameSize) {
		return 0, fmt.Errorf("%w: bodyLen=%d, maxFrameSize=%d", ErrFrameTooLarge, header.BodyLen, maxFrameSize)
	}
	return int(n), nil
}

func encodeFrame(flags uint8, body []byte) []byte {
	buf := make([]byte, headerSize+len(body))
	buf[0] = flags
	frameByteOrder.PutUint32(buf[1:headerSize], uint32(len(body)))
	copy(buf[headerSize:], body)
	return buf
}
