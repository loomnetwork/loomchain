package factory

import (
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts"
	chain "github.com/loomnetwork/loomchain/receipts/chain"
	`github.com/loomnetwork/loomchain/receipts/leveldb`
	
	//receipt_v2 "github.com/loomnetwork/loomchain/receipts/v2"
)

type ReceiptHandlerVersion int32

const (
	ReceiptHandlerChain            ReceiptHandlerVersion = 1
	ReceiptHandlerLevelDb           ReceiptHandlerVersion = 2
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

type ReceiptHandlerFactoryFunc func(loomchain.State, loomchain.EventHandler) receipts.ReceiptHandler
type ReadReceiptHandlerFactoryFunc func(loomchain.ReadOnlyState) receipts.ReadReceiptHandler

func NewReceiptHandlerFactory(v ReceiptHandlerVersion) (ReceiptHandlerFactoryFunc, error) {
	switch v {
	case ReceiptHandlerChain:
		return func(s loomchain.State,eh loomchain.EventHandler) receipts.ReceiptHandler {
			return &chain.WriteStateReceipts{s,eh}
		}, nil
	case ReceiptHandlerLevelDb:
		return func(s loomchain.State,eh loomchain.EventHandler) receipts.ReceiptHandler {
			return &leveldb.WriteLevelDbReceipts{s,eh}
		}, nil
	}
	return nil, receipts.ErrInvalidVersion
}


func NewReadReceiptHandlerFactory(v ReceiptHandlerVersion) (ReadReceiptHandlerFactoryFunc, error) {
	switch v {
	case ReceiptHandlerChain:
		return func(s loomchain.ReadOnlyState) receipts.ReadReceiptHandler {
			return &chain.ReadStateReceipts{s}
		}, nil
	case ReceiptHandlerLevelDb:
		return func(s loomchain.ReadOnlyState) receipts.ReadReceiptHandler {
			return &leveldb.ReadLevelDbReceipts{ s}
		}, nil
	}
	return nil, receipts.ErrInvalidVersion
}