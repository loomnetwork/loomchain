package loomchain

import (
	"github.com/loomnetwork/go-loom/plugin/types"

	"github.com/pkg/errors"
)

var (
	ErrInvalidVersion = errors.New("invalid receipt handler version")
)

type ReadReceiptHandler interface {
	GetReceipt(txHash []byte) (types.EvmTxReceipt, error)
	GetPendingReceipt(txHash []byte) (types.EvmTxReceipt, error)
	GetPendingTxHashList() [][]byte
	GetCurrentReceipt() *types.EvmTxReceipt
}

type ReceiptHandlerStore interface {
	CommitBlock(height int64) error
	CommitCurrentReceipt()
	DiscardCurrentReceipt()
	ClearData() error
	Close()
}

type ReceiptHandlerProvider interface {
	Store() ReceiptHandlerStore
	Reader() ReadReceiptHandler
	Writer() WriteReceiptHandler
}
