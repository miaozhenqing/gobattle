package service

import (
	"fmt"
	"gobattle/game/proto"
)

func dbFailed(err error, format string, args ...any) error {
	return fmt.Errorf("%w: %s: %w", proto.CodeDBFailed, fmt.Sprintf(format, args...), err)
}
