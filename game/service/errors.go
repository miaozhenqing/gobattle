package service

import (
	"errors"
	"fmt"
	"gobattle/btrouter"
	"gobattle/game/protocol"
)

func dbFailed(err error, format string, args ...any) error {
	return fmt.Errorf("%w: %s: %w", protocol.CodeDBFailed, fmt.Sprintf(format, args...), err)
}

// 事务返回的错误：fn 里返回的业务 Code 原样透出，其余（DB、提交失败）包装成 CodeDBFailed
func txFailed(err error, format string, args ...any) error {
	var code btrouter.Code
	if errors.As(err, &code) {
		return err
	}
	return dbFailed(err, format, args...)
}
