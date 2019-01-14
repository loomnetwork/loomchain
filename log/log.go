package log

import (
	"context"
	"io"
	"sync"

	kitlog "github.com/go-kit/kit/log"
	kitlevel "github.com/go-kit/kit/log/level"
	loom "github.com/loomnetwork/go-loom"
	tlog "github.com/tendermint/tendermint/libs/log"
)

// For compatibility with tendermint logger

type TMLogger tlog.Logger

var (
	NewTMLogger   = tlog.NewTMLogger
	NewTMFilter   = tlog.NewFilter
	TMAllowLevel  = tlog.AllowLevel
	NewSyncWriter = kitlog.NewSyncWriter
	Root          TMLogger
	Default       *loom.Logger
	DefaultJSON   *loom.Logger
	LevelKey      = kitlevel.Key()
)

var onceSetup sync.Once

func setupRootLogger(w io.Writer) {
	rootLoggerFunc := func(w io.Writer) TMLogger {
		return NewTMLogger(NewSyncWriter(w))
	}
	Root = rootLoggerFunc(w)
}

func setupLoomLogger(logLevel string, w io.Writer) {
	tlogTr := func(w io.Writer) kitlog.Logger {
		return tlog.NewTMFmtLogger(w)
	}
	jlogTr := func(w io.Writer) kitlog.Logger {
		return kitlog.NewJSONLogger(w)
	}
	DefaultJSON = loom.MakeLoomLogger(logLevel, w, jlogTr)
	Default = loom.MakeLoomLogger(logLevel, w, tlogTr)
}

func Setup(loomLogLevel, dest string) {
	onceSetup.Do(func() {
		w := loom.MakeFileLoggerWriter(loomLogLevel, dest)
		setupRootLogger(w)
		setupLoomLogger(loomLogLevel, w)
	})
}

// Info logs a message at level Debug.
func Info(msg string, keyvals ...interface{}) {
	Default.Info(msg, keyvals...)
}

//NewJSONLogger wrapper for each level for proper JSON format

func JSONInfo(msg string, keyvals ...interface{}) {
	DefaultJSON.Info(msg, keyvals...)
}

// Debug logs a message at level Debug.
func Debug(msg string, keyvals ...interface{}) {
	Default.Debug(msg, keyvals...)
}

func JSONDebug(msg string, keyvals ...interface{}) {
	DefaultJSON.Debug(msg, keyvals...)
}

// Error logs a message at level Error.
func Error(msg string, keyvals ...interface{}) {
	Default.Error(msg, keyvals...)
}

func JSONError(msg string, keyvals ...interface{}) {
	DefaultJSON.Error(msg, keyvals...)
}

// Warn logs a message at level Debug.
func Warn(msg string, keyvals ...interface{}) {
	Default.Warn(msg, keyvals...)
}

func JSONWarn(msg string, keyvals ...interface{}) {
	DefaultJSON.Warn(msg, keyvals...)
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

func Log(ctx context.Context) tlog.Logger {
	logger, _ := ctx.Value(contextKeyLog).(tlog.Logger)
	if logger == nil {
		return Root
	}

	return logger
}
