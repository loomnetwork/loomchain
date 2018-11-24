// +build evm

package query

import (
	"fmt"

	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
)

func QueryChain(state loomchain.ReadOnlyState, ethFilter eth.EthFilter, readReceipts loomchain.ReadReceiptHandler) ([]*ptypes.EthFilterLog, error) {
	start, err := eth.DecBlockHeight(state.Block().Height, eth.BlockHeight(ethFilter.FromBlock))
	if err != nil {
		return nil, err
	}
	end, err := eth.DecBlockHeight(state.Block().Height, eth.BlockHeight(ethFilter.ToBlock))
	if err != nil {
		return nil, err
	}

	return GetBlockLogRange(state, start, end, ethFilter.EthBlockFilter, readReceipts)
}

func DeprecatedQueryChain(query string, state loomchain.ReadOnlyState, readReceipts loomchain.ReadReceiptHandler) ([]byte, error) {
	ethFilter, err := utils.UnmarshalEthFilter([]byte(query))
	if err != nil {
		return nil, err
	}
	start, err := utils.DeprecatedBlockNumber(string(ethFilter.FromBlock), uint64(state.Block().Height))
	if err != nil {
		return nil, err
	}
	end, err := utils.DeprecatedBlockNumber(string(ethFilter.ToBlock), uint64(state.Block().Height))
	if err != nil {
		return nil, err
	}

	eventLogs, err := GetBlockLogRange(state, start, end, ethFilter.EthBlockFilter, readReceipts)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(&ptypes.EthFilterLogList{EthBlockLogs: eventLogs})
}

func GetBlockLogRange(
	state loomchain.ReadOnlyState,
	from, to uint64,
	ethFilter eth.EthBlockFilter,
	readReceipts loomchain.ReadReceiptHandler,
) ([]*ptypes.EthFilterLog, error) {
	if from > to {
		return nil, fmt.Errorf("to block before end block")
	}
	eventLogs := []*ptypes.EthFilterLog{}

	for height := from; height <= to; height++ {
		blockLogs, err := GetBlockLogs(state, ethFilter, height, readReceipts)
		if err != nil {
			return nil, err
		}
		eventLogs = append(eventLogs, blockLogs...)
	}
	return eventLogs, nil
}

func GetBlockLogs(
	state loomchain.ReadOnlyState,
	ethFilter eth.EthBlockFilter,
	height uint64,
	readReceipts loomchain.ReadReceiptHandler,
) ([]*ptypes.EthFilterLog, error) {
	bloomFilter := common.GetBloomFilter(state, height)
	if len(bloomFilter) > 0 {
		if MatchBloomFilter(ethFilter, bloomFilter) {
			txHashList, err := common.GetTxHashList(state, height)
			if err != nil {
				return nil, errors.Wrapf(err, "txhash for block height %d", height)
			}
			var logsBlock []*ptypes.EthFilterLog
			for _, txHash := range txHashList {
				txReceipt, err := readReceipts.GetReceipt(state, txHash)
				if err != nil {
					return nil, errors.Wrap(err, "getting receipt")
				}
				logsTx, err := getTxHashLogs(txReceipt, ethFilter, txHash)
				if err != nil {
					return nil, errors.Wrap(err, "logs for tx")
				}
				logsBlock = append(logsBlock, logsTx...)
			}
			return logsBlock, nil
		}
	}
	return nil, nil
}

func GetPendingBlockLogs(ethFilter eth.EthBlockFilter, receiptHandler loomchain.ReadReceiptHandler) ([]*ptypes.EthFilterLog, error) {
	txHashList := receiptHandler.GetPendingTxHashList()
	var logsBlock []*ptypes.EthFilterLog
	for _, txHash := range txHashList {
		txReceipt, err := receiptHandler.GetPendingReceipt(txHash)
		if err != nil {
			return nil, errors.Wrap(err, "cannot find pending tx receipt matching hash")
		}
		logsTx, err := getTxHashLogs(txReceipt, ethFilter, txHash)
		if err != nil {
			return nil, errors.Wrap(err, "logs for tx")
		}
		logsBlock = append(logsBlock, logsTx...)
	}
	return logsBlock, nil
}

func getTxHashLogs(txReceipt ptypes.EvmTxReceipt, filter eth.EthBlockFilter, txHash []byte) ([]*ptypes.EthFilterLog, error) {
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

func MatchBloomFilter(ethFilter eth.EthBlockFilter, bloomFilter []byte) bool {
	bFilter := bloom.NewBloomFilter()
	if len(ethFilter.Addresses) > 0 {
		found := false
		for _, addr := range ethFilter.Addresses {
			if bFilter.Contains(bloomFilter, []byte(addr)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}


	for _, topics := range ethFilter.Topics {
		if len(topics) > 0 {
			found := false
			for _, topic := range topics {
				if bFilter.Contains(bloomFilter, []byte(topic)) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}
