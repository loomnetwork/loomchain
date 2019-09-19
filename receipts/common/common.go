package common

import (
	"context"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	StatusTxSuccess = int32(1)
	StatusTxFail    = int32(0)
)

var (
	ErrTxReceiptNotFound      = errors.New("Tx receipt not found")
	ErrPendingReceiptNotFound = errors.New("Pending receipt not found")
)

func MockState(height uint64) loomchain.State {
	header := abci.Header{}
	header.Height = int64(height)
	return loomchain.NewStoreState(context.Background(), store.NewMemStore(), header, nil, nil)
}

func MockStateTx(state loomchain.State, height, TxNum uint64) loomchain.State {
	header := abci.Header{}
	header.Height = int64(height)
	header.NumTxs = int64(TxNum)
	return loomchain.NewStoreState(context.Background(), state, header, nil, nil)
}

func MockStateAt(state loomchain.State, newHeight uint64) loomchain.State {
	header := abci.Header{}
	header.Height = int64(newHeight)
	return loomchain.NewStoreState(context.Background(), state, header, nil, nil)
}
