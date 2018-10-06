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
}

type ReceiptHandler interface {
	SaveEventsAndHashReceipt(state State, caller, addr loom.Address, events []*EventData, err error) ([]byte, error)
	ClearData() error
	Close()

	ReadReceiptHandler
}

type ReceiptHandlerDBTx interface {
	BeginTx()
	Rollback()   //this is a noop if the commit already happened
	CommitFail() //stores the failed tx, but assigns do an error status
	Commit()

	ReceiptHandler
}
