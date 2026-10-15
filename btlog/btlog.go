package btlog

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
)

var ins *btLog
var once sync.Once

type btLog struct {
	logger *slog.Logger
}

func Ins() *btLog {
	once.Do(func() {
		ins = &btLog{
			logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			})),
		}
	})
	return ins
}

func (b *btLog) Debug(v ...interface{}) {
	b.logger.Debug(fmt.Sprint(v...))
}

func (b *btLog) Info(v ...interface{}) {
	b.logger.Info(fmt.Sprint(v...))
}

func (b *btLog) Warn(v ...interface{}) {
	b.logger.Warn(fmt.Sprint(v...))
}

func (b *btLog) Error(v ...interface{}) {
	b.logger.Error(fmt.Sprint(v...))
}

func (b *btLog) Debugf(format string, v ...interface{}) {
	b.logger.Debug(fmt.Sprintf(format, v...))
}

func (b *btLog) Infof(format string, v ...interface{}) {
	b.logger.Info(fmt.Sprintf(format, v...))
}

func (b *btLog) Warnf(format string, v ...interface{}) {
	b.logger.Warn(fmt.Sprintf(format, v...))
}

func (b *btLog) Errorf(format string, v ...interface{}) {
	b.logger.Error(fmt.Sprintf(format, v...))
}
