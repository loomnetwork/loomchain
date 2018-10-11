package loomchain

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/pkg/errors"
)

var (
	ReceiptPrefix = []byte("receipt")
	BloomPrefix   = []byte("bloomFilter")
	TxHashPrefix  = []byte("txHash")
	
	ErrInvalidVersion = errors.New("invalid receipt handler version")
)

const (
	StatusTxSuccess = int32(1)
	StatusTxFail    = int32(0)
)

type ReadReceiptHandler interface {
	GetReceipt(state ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error)
	GetPendingReceipt(txHash []byte) (types.EvmTxReceipt, error)
	GetPendingTxHashList() ([][]byte)
}

type ReceiptHandler interface {
	SetFailStatusCurrentReceipt()
	CommitBlock(state State, height int64) error
	CommitCurrentReceipt()
	ClearData() error
	ReadOnlyHandler() ReadReceiptHandler
	Close() error
}

type WriteReceiptHandler interface {
	CacheReceipt(state State, caller, addr loom.Address, events []*EventData, err error) ([]byte, error)
}