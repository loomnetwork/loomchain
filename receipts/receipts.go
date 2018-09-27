package receipts

import (
	`github.com/loomnetwork/go-loom`
	`github.com/loomnetwork/go-loom/plugin/types`
	`github.com/loomnetwork/loomchain`
	`github.com/pkg/errors`
)

var (
	ReceiptPrefix = []byte("receipt")
	BloomPrefix   = []byte("bloomFilter")
	TxHashPrefix  = []byte("txHash")
	
	ErrInvalidVersion    = errors.New("invalid receipt hanlder version")
)

type ReadReceiptHandler interface {
	GetReceipt(txHash []byte) (types.EvmTxReceipt, error)
	GetTxHash(height uint64) ([]byte, error)
	GetBloomFilter(height uint64) ([]byte, error)
}

type ReceiptHandler interface {
	SaveEventsAndHashReceipt(caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error)
	ClearData() error
}


