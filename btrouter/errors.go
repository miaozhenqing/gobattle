package btrouter

import (
	"fmt"
	"unicode/utf8"
)

const (
	CodeOK                uint32 = 0
	CodeInvalidArgument   uint32 = 3
	CodeResourceExhausted uint32 = 8
	CodeUnimplemented     uint32 = 12
	CodeInternal          uint32 = 13
	CodeUnauthenticated   uint32 = 16
)

// todo 按照模块前缀分
const (
	MinRequestMsgId uint32 = 1000
	MinPushMsgId    uint32 = 50000
)

const maxErrMsgLen = 256

// Error 业务错误
type Error struct {
	Code  uint32
	Msg   string
	cause error
}

func NewError(code uint32, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("code=%d msg=%s cause=%v", e.Code, e.Msg, e.cause)
	}
	return fmt.Sprintf("code=%d msg=%s", e.Code, e.Msg)
}

func (e *Error) Unwrap() error {
	return e.cause
}

func errInvalidArgument(cause error) *Error {
	return &Error{Code: CodeInvalidArgument, Msg: "invalid argument", cause: cause}
}

func errInternal(cause error) *Error {
	return &Error{Code: CodeInternal, Msg: "internal error", cause: cause}
}

var (
	errUnimplemented     = NewError(CodeUnimplemented, "unimplemented")
	errResourceExhausted = NewError(CodeResourceExhausted, "too many requests")
	errUnauthenticated   = NewError(CodeUnauthenticated, "unauthenticated")
)

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
