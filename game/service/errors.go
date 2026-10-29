package service

import (
	"fmt"
	"gobattle/game/protocol"
)

func dbFailed(err error, format string, args ...any) error {
	return fmt.Errorf("%w: %s: %w", protocol.CodeDBFailed, fmt.Sprintf(format, args...), err)
}
