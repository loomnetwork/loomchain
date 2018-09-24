package factory

import (
	"github.com/loomnetwork/loomchain"
	common "github.com/loomnetwork/loomchain/receipts"
	receipt_v1 "github.com/loomnetwork/loomchain/receipts/chain"
	//receipt_v2 "github.com/loomnetwork/loomchain/receipts/v2"
)

type ReceiptHandlerVersion int32

const (
	ReceiptHandlerChain            ReceiptHandlerVersion = 1
	ReceiptHandlerLocal            ReceiptHandlerVersion = 2
	DefaultReceiptHandlerVersion ReceiptHandlerVersion = ReceiptHandlerChain
)

// RegistryVersionFromInt safely converts an int to RegistryVersion.
func ReceiptHandlerVersionFromInt(v int32) (ReceiptHandlerVersion, error) {
	if v < 0 || v > int32(DefaultReceiptHandlerVersion) {
		return 0, common.ErrInvalidVersion
	}
	if v == 0 {
		return ReceiptHandlerChain, nil
	}
	return ReceiptHandlerVersion(v), nil
}

type ReceiptHandlerFactoryFunc func(loomchain.State, loomchain.EventHandler) common.ReceiptHandler

// NewRegistryFactory returns a factory function that can be used to create a Registry instance
// matching the specified version.
func NewReceiptHandlerFactory(v ReceiptHandlerVersion) (ReceiptHandlerFactoryFunc, error) {
	switch v {
	case ReceiptHandlerChain:
		return func(s loomchain.State,eh loomchain.EventHandler) common.ReceiptHandler {
			return &receipt_v1.StateReceipts{s,eh}
		}, nil
	//case ReceiptHandlerLocal:
	//	return func(s loomchain.State) common.Registry {
	//		return &receipt_v2.StateRegistry{state: s}
	//	}, nil
	}
	return nil, common.ErrInvalidVersion
}
