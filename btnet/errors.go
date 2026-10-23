package btnet

import (
	"errors"
	"fmt"
)

var (
	ErrConnClosed       = errors.New("conn is closed")
	ErrFrameTooLarge    = errors.New("frame too large")
	ErrUnsupportedFlags = errors.New("unsupported frame flags")
)

type PanicError struct {
	Value interface{}
	Stack []byte
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("handler panic: %v", e.Value)
}
