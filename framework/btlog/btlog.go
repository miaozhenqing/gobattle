package btlog

import (
	"log/slog"
	"os"
)

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	With(args ...any) Logger
}

// TextHandler（本地调试）
const textFormat = true

var ins Logger = &slogLogger{logger: slog.New(newHandler())}

func newHandler() slog.Handler {
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	if textFormat {
		return slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.NewJSONHandler(os.Stdout, opts)
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

func (b *slogLogger) Debug(msg string, args ...any) {
	b.logger.Debug(msg, args...)
}

func (b *slogLogger) Info(msg string, args ...any) {
	b.logger.Info(msg, args...)
}

func (b *slogLogger) Warn(msg string, args ...any) {
	b.logger.Warn(msg, args...)
}

func (b *slogLogger) Error(msg string, args ...any) {
	b.logger.Error(msg, args...)
}

func (b *slogLogger) With(args ...any) Logger {
	return &slogLogger{logger: b.logger.With(args...)}
}
