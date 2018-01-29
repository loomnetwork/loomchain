package loom

import (
	"context"

	abci "github.com/tendermint/abci/types"
)

type TxHandler interface {
	Handle(ctx context.Context, txBytes []byte) error
}

type TxHandlerFunc func(ctx context.Context, txBytes []byte) error

func (f TxHandlerFunc) Handle(ctx context.Context, txBytes []byte) error {
	return f(ctx, txBytes)
}

type Application struct {
	abci.BaseApplication

	TxHandler
}

var _ abci.Application = &Application{}

func (a *Application) CheckTx(txBytes []byte) abci.ResponseCheckTx {
	if len(txBytes) == 0 {
		return abci.ResponseCheckTx{Code: 1, Log: "transaction empty"}
	}
	return abci.ResponseCheckTx{Code: abci.CodeTypeOK}
}

func (a *Application) DeliverTx(txBytes []byte) abci.ResponseDeliverTx {
	ctx := context.Background()
	err := a.TxHandler.Handle(ctx, txBytes)
	if err != nil {
		return abci.ResponseDeliverTx{Code: 1, Log: err.Error()}
	}
	return abci.ResponseDeliverTx{Code: abci.CodeTypeOK}
}
