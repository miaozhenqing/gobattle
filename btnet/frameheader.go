package btnet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

type frameHeader struct {
	MessageID uint32
	DataLen   uint32
}

// msgId(4) + dataLen(4)
const headerSize = 8

// 这里做一个拦截
func init() {
	if binary.Size(frameHeader{}) != headerSize {
		panic(fmt.Sprintf("frameHeader size %d mismatch headerSize %d", binary.Size(frameHeader{}), headerSize))
	}
}

var frameByteOrder binary.ByteOrder = binary.BigEndian

var errShortFrameHeader = errors.New("short frame header")

func encodeFrameHeader(header frameHeader) ([]byte, error) {
	var buf bytes.Buffer
	if err := binary.Write(&buf, frameByteOrder, header); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeFrameHeader(data []byte) (frameHeader, error) {
	if len(data) < headerSize {
		return frameHeader{}, errShortFrameHeader
	}

	var header frameHeader
	err := binary.Read(bytes.NewReader(data[:headerSize]), frameByteOrder, &header)
	return header, err
}

func frameLen(header frameHeader, maxFrameSize int) (int, error) {
	n := int64(headerSize) + int64(header.DataLen)
	if n > int64(maxFrameSize) {
		return 0, fmt.Errorf("%w: dataLen=%d, maxFrameSize=%d", ErrFrameTooLarge, header.DataLen, maxFrameSize)
	}
	return int(n), nil
}
