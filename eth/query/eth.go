// +build evm

package query

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/store"
)

func QueryChain(query string, state loomchain.ReadOnlyState) ([]byte, error) {
	ethFilter, err := utils.UnmarshalEthFilter([]byte(query))
	if err != nil {
		return nil, err
	}
	start, err := utils.BlockNumber(ethFilter.FromBlock, uint64(state.Block().Height))
	if err != nil {
		return nil, err
	}
	end, err := utils.BlockNumber(ethFilter.ToBlock, uint64(state.Block().Height))
	if err != nil {
		return nil, err
	}

	eventLogs, err := GetBlockLogRange(start, end, ethFilter.EthBlockFilter, state)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(&ptypes.EthFilterLogList{EthBlockLogs: eventLogs})
}

func GetBlockLogRange(from, to uint64, ethFilter utils.EthBlockFilter, state loomchain.ReadOnlyState) ([]*ptypes.EthFilterLog, error) {
	if from > to {
		return nil, fmt.Errorf("to block before end block")
	}
	eventLogs := []*ptypes.EthFilterLog{}

	for height := from; height <= to; height++ {
		blockLogs, err := GetBlockLogs(ethFilter, state, height)
		if err != nil {
			return nil, err
		}
		eventLogs = append(eventLogs, blockLogs...)
	}
	return eventLogs, nil
}

func GetBlockLogs(ethFilter utils.EthBlockFilter, state loomchain.ReadOnlyState, height uint64) ([]*ptypes.EthFilterLog, error) {
	heightB := utils.BlockHeightToBytes(height)
	bloomState := store.PrefixKVReader(utils.BloomPrefix, state)
	bloomFilter := bloomState.Get(heightB)
	if len(bloomFilter) > 0 {
		if MatchBloomFilter(ethFilter, bloomFilter) {
			txHashState := store.PrefixKVReader(utils.TxHashPrefix, state)
			txHash := txHashState.Get(heightB)
			return getTxHashLogs(state, ethFilter, txHash)
		}
	}
	return nil, nil
}

func getTxHashLogs(state loomchain.ReadOnlyState, filter utils.EthBlockFilter, txHash []byte) ([]*ptypes.EthFilterLog, error) {
	receiptState := store.PrefixKVReader(utils.ReceiptPrefix, state)
	txReceiptProto := receiptState.Get(txHash)
	txReceipt := ptypes.EvmTxReceipt{}
	err := proto.Unmarshal(txReceiptProto, &txReceipt)
	if err != nil {
		return nil, err
	}
	var blockLogs []*ptypes.EthFilterLog

	for i, eventLog := range txReceipt.Logs {
		if utils.MatchEthFilter(filter, *eventLog) {
			var topics [][]byte
			for _, topic := range eventLog.Topics {
				topics = append(topics, []byte(topic))
			}
			blockLogs = append(blockLogs, &ptypes.EthFilterLog{
				Removed:          false,
				LogIndex:         int64(i),
				TransactionIndex: txReceipt.TransactionIndex,
				TransactionHash:  txHash,
				BlockHash:        txReceipt.BlockHash,
				BlockNumber:      txReceipt.BlockNumber,
				Address:          eventLog.Address.Local,
				Data:             eventLog.EncodedBody,
				Topics:           topics,
			})
		}
	}
	return blockLogs, nil
}

func MatchBloomFilter(ethFilter utils.EthBlockFilter, bloomFilter []byte) bool {
	bFilter := NewBloomFilter()
	for _, addr := range ethFilter.Addresses {
		if !bFilter.Contains(bloomFilter, []byte(addr)) {
			return false
		}
	}
	for _, topics := range ethFilter.Topics {
		for _, topic := range topics {
			if !bFilter.Contains(bloomFilter, []byte(topic)) {
				return false
			}
		}
	}
	return true
}
