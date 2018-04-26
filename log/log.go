package log

import (
	"context"
	"fmt"
	"os"

	kitlog "github.com/go-kit/kit/log"
	tlog "github.com/tendermint/tmlibs/log"
)

// Reexported types
type Logger = tlog.Logger

var (
	NewLogger     = tlog.NewTMLogger
	NewSyncWriter = kitlog.NewSyncWriter
	Root          = NewLogger(NewSyncWriter(os.Stdout))
)

type contextKey string

func (c contextKey) String() string {
	return "log " + string(c)
}

var (
	contextKeyLog = contextKey("log")
)

func SetContext(ctx context.Context, log Logger) context.Context {
	return context.WithValue(ctx, contextKeyLog, log)
}

func Log(ctx context.Context) Logger {
	logger, _ := ctx.Value(contextKeyLog).(Logger)
	if logger == nil {
		return Root
	}

	return logger
}

var TxMiddleware = loom.TxMiddlewareFunc(func(
	state loom.State,
	txBytes []byte,
	next loom.TxHandlerFunc,
) (loom.TxHandlerResult, error) {
	// TODO: set some tx specific logging info
	return next(state, txBytes)
})

var PostCommitMiddleware = loom.PostCommitMiddlewareFunc(func(
	state loom.State,
	txBytes []byte,
	res loom.TxHandlerResult,
	next loom.PostCommitHandler,
) error {
	Root.Debug("Running post commit logger")
	Root.Info(string(txBytes))
	Root.Info(fmt.Sprintf("%+v", res))
	return next(state, txBytes, res)
})
