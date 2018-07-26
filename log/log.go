package log

import (
	"context"
	"io"
	"sync"

	kitlog "github.com/go-kit/kit/log"
	kitlevel "github.com/go-kit/kit/log/level"
	loom "github.com/loomnetwork/go-loom"
	tlog "github.com/tendermint/tmlibs/log"
)

type TMLogger tlog.Logger

var (
	NewSyncWriter = kitlog.NewSyncWriter
	Default       loom.ILogger
	LevelKey      = kitlevel.Key()
)

var onceSetup sync.Once

func setupLoomLogger(logLevel string, w io.Writer) {
}

func Setup(loomLogLevel, dest string) {
	onceSetup.Do(func() {
		Default = loom.NewLoomLogger(loomLogLevel, dest)
	})
}

// Info logs a message at level Debug.
func Info(msg string, keyvals ...interface{}) {
	Default.Info(msg, keyvals...)
}

// Debug logs a message at level Debug.
func Debug(msg string, keyvals ...interface{}) {
	Default.Debug(msg, keyvals...)
}

// Error logs a message at level Error.
func Error(msg string, keyvals ...interface{}) {
	Default.Error(msg, keyvals...)
}

type contextKey string

func (c contextKey) String() string {
	return "log " + string(c)
}

var (
	contextKeyLog = contextKey("log")
)

func SetContext(ctx context.Context, log loom.Logger) context.Context {
	return context.WithValue(ctx, contextKeyLog, log)
}

func Log(ctx context.Context) loom.ILogger {
	logger, _ := ctx.Value(contextKeyLog).(loom.ILogger)
	if logger == nil {
		return Default
	}

	return logger
}
