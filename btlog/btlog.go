package btlog

import (
	"fmt"
	"log/slog"
	"os"
)

type Logger interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warn(v ...interface{})
	Error(v ...interface{})

	Debugf(format string, v ...interface{})
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Errorf(format string, v ...interface{})
}

var ins Logger = &slogLogger{
	logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})),
}

func Ins() Logger {
	return ins
}

func SetLogger(l Logger) {
	if l == nil {
		panic("logger is nil")
	}
	ins = l
}

type slogLogger struct {
	logger *slog.Logger
}

func (b *slogLogger) Debug(v ...interface{}) {
	b.logger.Debug(fmt.Sprint(v...))
}

func (b *slogLogger) Info(v ...interface{}) {
	b.logger.Info(fmt.Sprint(v...))
}

func (b *slogLogger) Warn(v ...interface{}) {
	b.logger.Warn(fmt.Sprint(v...))
}

func (b *slogLogger) Error(v ...interface{}) {
	b.logger.Error(fmt.Sprint(v...))
}

func (b *slogLogger) Debugf(format string, v ...interface{}) {
	b.logger.Debug(fmt.Sprintf(format, v...))
}

func (b *slogLogger) Infof(format string, v ...interface{}) {
	b.logger.Info(fmt.Sprintf(format, v...))
}

func (b *slogLogger) Warnf(format string, v ...interface{}) {
	b.logger.Warn(fmt.Sprintf(format, v...))
}

func (b *slogLogger) Errorf(format string, v ...interface{}) {
	b.logger.Error(fmt.Sprintf(format, v...))
}
