package log

import (
	"context"
	"os"

	kitlog "github.com/go-kit/kit/log"
	kitlevel "github.com/go-kit/kit/log/level"
	tlog "github.com/tendermint/tmlibs/log"
)

// For compatibility with tendermint logger
var msgKey = "_msg"

type TMLogger tlog.Logger

type Logger struct {
	kitlog.Logger
}

// Info logs a message at level Info.
func (l *Logger) Info(msg string, keyvals ...interface{}) {
	lWithLevel := kitlevel.Info(l)
	if err := kitlog.With(lWithLevel, msgKey, msg).Log(keyvals...); err != nil {
		errLogger := kitlevel.Error(l)
		kitlog.With(errLogger, msgKey, msg).Log("err", err)
	}
}

// Debug logs a message at level Debug.
func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	lWithLevel := kitlevel.Debug(l)
	if err := kitlog.With(lWithLevel, msgKey, msg).Log(keyvals...); err != nil {
		errLogger := kitlevel.Error(l)
		errLogger.Log("err", err)
	}
}

// Error logs a message at level Error.
func (l *Logger) Error(msg string, keyvals ...interface{}) {
	lWithLevel := kitlevel.Error(l)
	lWithMsg := kitlog.With(lWithLevel, msgKey, msg)
	if err := lWithMsg.Log(keyvals...); err != nil {
		lWithMsg.Log("err", err)
	}
}

// Warn logs a message at level Debug.
func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	lWithLevel := kitlevel.Warn(l)
	if err := kitlog.With(lWithLevel, msgKey, msg).Log(keyvals...); err != nil {
		errLogger := kitlevel.Error(l)
		kitlog.With(errLogger, msgKey, msg).Log("err", err)
	}
}

var (
	NewTMLogger   = tlog.NewTMLogger
	NewTMFilter   = tlog.NewFilter
	TMAllowLevel  = tlog.AllowLevel
	NewSyncWriter = kitlog.NewSyncWriter
	Root          = NewTMLogger(NewSyncWriter(os.Stderr))
	NewFilter     = func(next kitlog.Logger, options ...kitlevel.Option) *Logger {
		return &Logger{kitlevel.NewFilter(next, options...)}
	}
	Default = &Logger{
		tlog.NewTMFmtLogger(os.Stderr),
	}
	LevelKey = kitlevel.Key()
)

type contextKey string

func (c contextKey) String() string {
	return "log " + string(c)
}

var (
	contextKeyLog = contextKey("log")
)

var (
	AllowDebug = kitlevel.AllowDebug
	AllowInfo  = kitlevel.AllowInfo
	AllowWarn  = kitlevel.AllowWarn
	AllowError = kitlevel.AllowError
	Allow      = func(level string) kitlevel.Option {
		switch level {
		case "debug":
			return AllowDebug()
		case "info":
			return AllowInfo()
		case "warn":
			return AllowWarn()
		case "error":
			return AllowError()
		default:
			return nil
		}
	}
)

func SetContext(ctx context.Context, log Logger) context.Context {
	return context.WithValue(ctx, contextKeyLog, log)
}

func Log(ctx context.Context) tlog.Logger {
	logger, _ := ctx.Value(contextKeyLog).(tlog.Logger)
	if logger == nil {
		return Root
	}

	return logger
}

func ContractLogger(name string) *Logger {
	return &Logger{kitlog.With(Default, "contract", name)}
}
