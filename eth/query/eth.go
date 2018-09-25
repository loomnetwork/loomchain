// +build evm

package query

import (
	"fmt"
	`github.com/loomnetwork/loomchain/eth/bloom`
	`github.com/pkg/errors`
	
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/receipts"
)

func QueryChain(query string, state loomchain.ReadOnlyState, readReceipts receipts.ReadReceiptHandler) ([]byte, error) {
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

	eventLogs, err := GetBlockLogRange(start, end, ethFilter.EthBlockFilter, readReceipts)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(&ptypes.EthFilterLogList{EthBlockLogs: eventLogs})
}

func GetBlockLogRange(
		from, to uint64,
		ethFilter utils.EthBlockFilter,
		readReceipts receipts.ReadReceiptHandler,
	) ([]*ptypes.EthFilterLog, error) {
	if from > to {
		return nil, fmt.Errorf("to block before end block")
	}
	eventLogs := []*ptypes.EthFilterLog{}

	for height := from; height <= to; height++ {
		blockLogs, err := GetBlockLogs(ethFilter, height, readReceipts)
		if err != nil {
			return nil, err
		}
		eventLogs = append(eventLogs, blockLogs...)
	}
	return eventLogs, nil
}

func GetBlockLogs(
		ethFilter utils.EthBlockFilter,
		height uint64,
		readReceipts receipts.ReadReceiptHandler,
	) ([]*ptypes.EthFilterLog, error) {
	bloomFilter, err := readReceipts.GetBloomFilter(height)
	if err != nil {
		return nil, errors.Wrapf(err, "getting bloom filter for height %d", height)
	}
	
	if len(bloomFilter) > 0 {
		if MatchBloomFilter(ethFilter, bloomFilter) {
			txHash, err := readReceipts.GetTxHash(height)
			if err != nil {
				return nil, errors.Wrapf(err, "getting txhash for height %d", height)
			}
			return getTxHashLogs(readReceipts, ethFilter, txHash)
		}
	}
	return nil, nil
}

func getTxHashLogs(readReceipts receipts.ReadReceiptHandler, filter utils.EthBlockFilter, txHash []byte) ([]*ptypes.EthFilterLog, error) {
	txReceipt, err := readReceipts.GetReceipt(txHash)
	if err != nil {
		return nil, errors.Wrap(err,"read receipt")
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
	bFilter := bloom.NewBloomFilter()
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
