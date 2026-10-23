package btrouter

import (
	"errors"
	"strconv"
)

type Code uint32

func (c Code) Error() string {
	return "code=" + strconv.FormatUint(uint64(c), 10)
}

const (
	CodeOK                Code = 0
	CodeInvalidArgument   Code = 3
	CodeResourceExhausted Code = 8
	CodeUnimplemented     Code = 12
	CodeInternal          Code = 13
	CodeUnauthenticated   Code = 16
)

var (
	ErrClientClosed    = errors.New("client closed")
	ErrInvalidEnvelope = errors.New("invalid envelope")
)
