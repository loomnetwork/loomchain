package log

import (
	"context"
	"io"
	"sync"

	kitlog "github.com/go-kit/kit/log"
	kitlevel "github.com/go-kit/kit/log/level"
	loom "github.com/loomnetwork/go-loom"
	//	kitsyslog "github.com/go-kit/kit/log/syslog"
	tlog "github.com/tendermint/tmlibs/log"
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
	Default = loom.MakeLoomLogger(logLevel, w, tlogTr)
}

func Setup(loomLogLevel, dest string) {
	onceSetup.Do(func() {
		w := loom.MakeFileLoggerWriter(loomLogLevel, dest)
		setupRootLogger(w)
		setupLoomLogger(loomLogLevel, w)
	})
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

//func ContractLogger(name string) *loom.Logger {
//	return &Logger{kitlog.With(Default, "contract", name)}
//}
