package loomchain

import (
	`github.com/loomnetwork/go-loom`
	`github.com/loomnetwork/go-loom/plugin/types`
	`github.com/pkg/errors`
)

var (
	ReceiptPrefix = []byte("receipt")
	BloomPrefix   = []byte("bloomFilter")
	TxHashPrefix  = []byte("txHash")
	
	ErrInvalidVersion    = errors.New("invalid receipt hanlder version")

)

const (
	StatusTxSuccess = int32(1)
	StatusTxFail = int32(0)
)

// Called from evm
type WriteReceiptCache interface {
	SaveEventsAndHashReceipt(state State, caller, addr loom.Address, events []*EventData, err error) ([]byte, error)
	Empty()
}

type ReadReceiptCache interface {
	GetReceipt() types.EvmTxReceipt
}

// Called from rpc.QueryServer
type ReadReceiptHandler interface {
	GetReceipt(txHash []byte) (types.EvmTxReceipt, error)
}

// Called from processTx
type WriteReceiptHandler interface {
	Commit(types.EvmTxReceipt) error
}

type WriteReceiptHandlerFactoryFunc func(State) (WriteReceiptHandler, error)
type ReadReceiptHandlerFactoryFunc func(State) (ReadReceiptHandler, error)

type ReceiptPlant interface {
	ReadCache() *ReadReceiptCache
	WriteCache() *WriteReceiptCache
	ReceiptReaderFactory() ReadReceiptHandlerFactoryFunc
	ReciepWriterFactory() WriteReceiptHandlerFactoryFunc
	CommitBloomFilters(state State, height uint64) error
}


