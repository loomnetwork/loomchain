package factory

import (
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts"
	"github.com/loomnetwork/loomchain/receipts/chain"
	"github.com/loomnetwork/loomchain/receipts/leveldb"
)

type ReceiptHandlerVersion int32

const (
	ReceiptHandlerChain          ReceiptHandlerVersion = 1
	ReceiptHandlerLevelDb        ReceiptHandlerVersion = 2
	DefaultReceiptHandlerVersion ReceiptHandlerVersion = ReceiptHandlerChain
)

func ReceiptHandlerVersionFromInt(v int32) (ReceiptHandlerVersion, error) {
	if v < 0 || v > int32(ReceiptHandlerLevelDb) {
		return 0, receipts.ErrInvalidVersion
	}
	if v == 0 {
		return ReceiptHandlerChain, nil
	}
	return ReceiptHandlerVersion(v), nil
}

//Allows runtime swapping of receipt handlers
type ReceiptHandlerFactory struct {
	v               ReceiptHandlerVersion
	chainReceipts   *chain.WriteStateReceipts
	leveldbReceipts *leveldb.WriteLevelDbReceipts
}

func (r *ReceiptHandlerFactory) GetTxHash(state loomchain.ReadOnlyState, height uint64) ([]byte, error) {
	switch r.v {
	case ReceiptHandlerChain:
		return r.chainReceipts.GetTxHash(state, height)
	case ReceiptHandlerLevelDb:
		return r.leveldbReceipts.GetTxHash(state, height)
	}
	return nil, receipts.ErrInvalidVersion
}

func (r *ReceiptHandlerFactory) GetBloomFilter(state loomchain.ReadOnlyState, height uint64) ([]byte, error) {
	switch r.v {
	case ReceiptHandlerChain:
		return r.chainReceipts.GetBloomFilter(state, height)
	case ReceiptHandlerLevelDb:
		return r.leveldbReceipts.GetBloomFilter(state, height)
	}
	return nil, receipts.ErrInvalidVersion
}

func (r *ReceiptHandlerFactory) GetReceipt(state loomchain.ReadOnlyState, txHash []byte) (types.EvmTxReceipt, error) {
	switch r.v {
	case ReceiptHandlerChain:
		return r.chainReceipts.GetReceipt(state, txHash)
	case ReceiptHandlerLevelDb:
		return r.leveldbReceipts.GetReceipt(state, txHash)
	}
	return types.EvmTxReceipt{}, receipts.ErrInvalidVersion
}

func (r *ReceiptHandlerFactory) Close() {
	switch r.v {
	case ReceiptHandlerChain:
		r.chainReceipts.Close()
	case ReceiptHandlerLevelDb:
		r.leveldbReceipts.Close()
	}
}

func (r *ReceiptHandlerFactory) ClearData() error {
	switch r.v {
	case ReceiptHandlerChain:
		return r.chainReceipts.ClearData()
	case ReceiptHandlerLevelDb:
		return r.leveldbReceipts.ClearData()
	}
	return receipts.ErrInvalidVersion
}

func (r *ReceiptHandlerFactory) SaveEventsAndHashReceipt(state loomchain.State, caller, addr loom.Address, events []*loomchain.EventData, err error) ([]byte, error) {
	switch r.v {
	case ReceiptHandlerChain:
		return r.chainReceipts.SaveEventsAndHashReceipt(state, caller, addr, events, err)
	case ReceiptHandlerLevelDb:
		return r.leveldbReceipts.SaveEventsAndHashReceipt(state, caller, addr, events, err)
	}
	return nil, receipts.ErrInvalidVersion
}

func NewReceiptHandlerFactory(v ReceiptHandlerVersion, eh loomchain.EventHandler) (receipts.ReceiptHandler, error) {
	r := &ReceiptHandlerFactory{v: v}
	switch r.v {
	case ReceiptHandlerChain:
		wsr := &chain.WriteStateReceipts{eh}
		r.chainReceipts = wsr
		return r, nil
	case ReceiptHandlerLevelDb:
		ldbr, err := leveldb.NewWriteLevelDbReceipts(eh)
		r.leveldbReceipts = ldbr
		return r, err
	}
	return nil, receipts.ErrInvalidVersion
}
