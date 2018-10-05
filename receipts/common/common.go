package common

import (
	`crypto/sha256`
	`encoding/binary`
	"github.com/gogo/protobuf/proto"
	`github.com/loomnetwork/go-loom`
	`github.com/loomnetwork/go-loom/plugin/types`
	`github.com/loomnetwork/loomchain`
	`github.com/loomnetwork/loomchain/eth/bloom`
	registry "github.com/loomnetwork/loomchain/registry/factory"
	`github.com/pkg/errors`
)

func WriteReceipt(
		state loomchain.State,
		caller, addr loom.Address,
		events []*loomchain.EventData,
		err error,
		eventHadler loomchain.EventHandler,
	) (types.EvmTxReceipt, error) {
	var status int32
	if err == nil {
		status = 1
	} else {
		status = 0
	}
	block := state.Block()
	txReceipt := types.EvmTxReceipt{
		TransactionIndex:  state.Block().NumTxs,
		BlockHash:         block.GetLastBlockID().Hash,
		BlockNumber:       state.Block().Height,
		CumulativeGasUsed: 0,
		GasUsed:           0,
		ContractAddress:   addr.Local,
		LogsBloom:         bloom.GenBloomFilter(events),
		Status:            status,
		CallerAddress:     caller.MarshalPB(),
	}
	
	preTxReceipt, errMarshal := proto.Marshal(&txReceipt)
	if errMarshal != nil {
		if err == nil {
			return types.EvmTxReceipt{}, errors.Wrap(errMarshal, "marhsal tx receipt")
		} else {
			return types.EvmTxReceipt{}, errors.Wrapf(err, "marshalling reciept err %v", errMarshal)
		}
	}
	h := sha256.New()
	h.Write(preTxReceipt)
	txHash := h.Sum(nil)
	
	txReceipt.TxHash = txHash
	blockHeight := uint64(txReceipt.BlockNumber)
	for _, event := range events {
		event.TxHash = txHash
		if eventHadler != nil {
			_ = eventHadler.Post(blockHeight, event)
		}
		pEvent := types.EventData(*event)
		txReceipt.Logs = append(txReceipt.Logs, &pEvent)
	}

	return txReceipt, nil
}

func BlockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightB, height)
	return heightB
}

func GetConfigContractAddress(state loomchain.State, createRegistry  registry.RegistryFactoryFunc) (loom.Address, error) {
	registryObject := createRegistry(state)
	configContractAddress, err := registryObject.Resolve("config")
	if err != nil {
		return loom.Address{}, errors.Wrap(err, "resolving config address")
	}
	return configContractAddress, nil
}

func GetConfignState(state loomchain.State, configContractAddress loom.Address) (loomchain.State,) {
	return loomchain.StateWithPrefix(loom.DataPrefix(configContractAddress), state)
}