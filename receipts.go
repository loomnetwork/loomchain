package loomchain

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/pkg/errors"
)

var (
	ErrInvalidVersion = errors.New("invalid receipt handler version")
)

type ReadReceiptHandler interface {
	GetReceipt(state ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error)
	GetPendingReceipt(txHash []byte) (types.EvmTxReceipt, error)
	GetPendingTxHashList() [][]byte
	GetCurrentReceipt() *types.EvmTxReceipt
}

type ReceiptHandlerStore interface {
	SetFailStatusCurrentReceipt()
	CommitBlock(state State, height int64) error
	CommitCurrentReceipt()
	DiscardCurrentReceipt()
	ClearData() error
	ReadOnlyHandler() ReadReceiptHandler
	Close() error
}

type WriteReceiptHandler interface {
	CacheReceipt(state State, caller, addr loom.Address, events []*EventData, err error) ([]byte, error)
}

type ReceiptHandlerProvider interface {
	StoreAt(blockHeight int64) (ReceiptHandlerStore, error)
	ReaderAt(blockHeight int64) (ReadReceiptHandler, error)
	WriterAt(blockHeight int64) (WriteReceiptHandler, error)
}
