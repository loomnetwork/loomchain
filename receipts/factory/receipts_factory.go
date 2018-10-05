package factory

import (
	`github.com/gogo/protobuf/proto`
	`github.com/loomnetwork/go-loom`
	ctypes `github.com/loomnetwork/go-loom/builtin/types/config`
	"github.com/loomnetwork/loomchain"
	`github.com/loomnetwork/loomchain/builtin/plugins/config`
	"github.com/loomnetwork/loomchain/receipts"
	"github.com/loomnetwork/loomchain/receipts/chain"
	`github.com/loomnetwork/loomchain/receipts/common`
	`github.com/loomnetwork/loomchain/receipts/leveldb`
	registry "github.com/loomnetwork/loomchain/registry/factory"
	`github.com/pkg/errors`
)

const (
	DefaultReceiptStorage = ctypes.ReceiptStorage_CHAIN
)

func ReceiptHandlerVersionFromInt(v int32) (ctypes.ReceiptStorage, error) {
	if v < 0 || v > int32(ctypes.ReceiptStorage_LEVELDB) {
		return 0, receipts.ErrInvalidVersion
	}
	if v == 0 {
		return ctypes.ReceiptStorage_CHAIN, nil
	}
	return ctypes.ReceiptStorage(v), nil
}

type ReceiptHandlerFactoryFunc func(loomchain.State, loomchain.EventHandler) (receipts.ReceiptHandler, error)
type ReadReceiptHandlerFactoryFunc func(loomchain.State) (receipts.ReadReceiptHandler, error)

func NewReceiptHandlerFactory(v ctypes.ReceiptStorage) (ReceiptHandlerFactoryFunc, error) {
	switch v {
	case ctypes.ReceiptStorage_CHAIN:
		return func(s loomchain.State,eh loomchain.EventHandler) (receipts.ReceiptHandler, error) {
			return &chain.WriteStateReceipts{s,eh}, nil
		}, nil
	case ctypes.ReceiptStorage_LEVELDB:
		return func(s loomchain.State,eh loomchain.EventHandler) (receipts.ReceiptHandler, error) {
			return &leveldb.WriteLevelDbReceipts{s,eh,}, nil
		}, nil
	}
	return nil, receipts.ErrInvalidVersion
}

func NewStateReceiptHandlerFactory(createRegistry  registry.RegistryFactoryFunc) (ReceiptHandlerFactoryFunc, error) {
	var configContractAddress loom.Address
	return func(s loomchain.State,eh loomchain.EventHandler) (receipts.ReceiptHandler, error) {
		if (0 == configContractAddress.Compare(loom.Address{})) {
			var err error
			configContractAddress, err = common.GetConfigContractAddress(s, createRegistry)
			if err != nil {
				return nil, errors.Wrap(err, "get contract address")
			}
		}
		configState := common.GetConfignState(s, configContractAddress)
		protoValue := configState.Get(config.StateKey(config.ConfigKeyRecieptStrage))
		value := ctypes.Value{}
		if err := proto.Unmarshal(protoValue, &value); err != nil {
			return nil, errors.Wrap(err ,"unmarshal config value")
		}
		switch value.GetReceiptStorage() {
			case ctypes.ReceiptStorage_CHAIN: return &chain.WriteStateReceipts{s,eh}, nil
			case ctypes.ReceiptStorage_LEVELDB:	return &leveldb.WriteLevelDbReceipts{s,eh,}, nil
			default: return nil, errors.Errorf("unrecognises receipt storage method, %v", value.GetReceiptStorage())
		}
	}, nil
}

func NewReadReceiptHandlerFactory(v ctypes.ReceiptStorage) (ReadReceiptHandlerFactoryFunc, error) {
	switch v {
	case ctypes.ReceiptStorage_CHAIN:
		return func(s loomchain.State) (receipts.ReadReceiptHandler, error) {
			return &chain.ReadStateReceipts{s}, nil
		}, nil
	case ctypes.ReceiptStorage_LEVELDB:
		return func(s loomchain.State) (receipts.ReadReceiptHandler, error) {
			return &leveldb.ReadLevelDbReceipts{ s}, nil
		}, nil
	}
	return nil, receipts.ErrInvalidVersion
}

func NewStateReadReceiptHandlerFactory(createRegistry  registry.RegistryFactoryFunc) (ReadReceiptHandlerFactoryFunc, error) {
	var configContractAddress loom.Address
	return func(s loomchain.State) (receipts.ReadReceiptHandler, error) {
		if (0 == configContractAddress.Compare(loom.Address{})) {
			var err error
			configContractAddress, err = common.GetConfigContractAddress(s, createRegistry)
			if err != nil {
				return nil, errors.Wrap(err, "get contract address")
			}
		}
		
		configState := common.GetConfignState(s, configContractAddress)
		protoValue := configState.Get(config.StateKey(config.ConfigKeyRecieptStrage))
		value := ctypes.Value{}
		if err := proto.Unmarshal(protoValue, &value); err != nil {
			return nil, errors.Wrap(err ,"unmarshal config value")
		}
		switch value.GetReceiptStorage() {
			case ctypes.ReceiptStorage_CHAIN: return &chain.ReadStateReceipts{s}, nil
			case ctypes.ReceiptStorage_LEVELDB:	return &leveldb.ReadLevelDbReceipts{ s}, nil
			default: return nil, errors.Errorf("unrecognises receipt storage method, %v", value.GetReceiptStorage())
		}
	}, nil
}

