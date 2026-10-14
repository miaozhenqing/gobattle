package btnet

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type frameHeader struct {
	MessageID uint32
	DataLen   uint32
}

var headerSize = binary.Size(frameHeader{})

// Frame fields use network byte order; changing this breaks existing frames.
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
