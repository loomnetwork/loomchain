// +build evm

package query

import (
	"fmt"

	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/rpc/eth"
	"github.com/loomnetwork/loomchain/store"
	"github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

func QueryChain(
	blockStore store.BlockStore, state loomchain.ReadOnlyState, ethFilter eth.EthFilter,
	readReceipts loomchain.ReadReceiptHandler, evmAuxStore *evmaux.EvmAuxStore, maxBlockRange uint64,
) ([]*ptypes.EthFilterLog, error) {
	start, err := eth.DecBlockHeight(state.Block().Height, eth.BlockHeight(ethFilter.FromBlock))
	if err != nil {
		return nil, err
	}
	end, err := eth.DecBlockHeight(state.Block().Height, eth.BlockHeight(ethFilter.ToBlock))
	if err != nil {
		return nil, err
	}
	if end < start {
		return nil, errors.New("invalid block range")
	}

	if end-start > maxBlockRange {
		return nil, fmt.Errorf("max allowed block range (%d) exceeded", maxBlockRange)
	}

	return GetBlockLogRange(blockStore, state, start, end, ethFilter.EthBlockFilter, readReceipts, evmAuxStore)
}

func DeprecatedQueryChain(
	query string, blockStore store.BlockStore, state loomchain.ReadOnlyState,
	readReceipts loomchain.ReadReceiptHandler, evmAuxStore *evmaux.EvmAuxStore, maxBlockRange uint64,
) ([]byte, error) {

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

	if end < start {
		return nil, errors.New("invalid block range")
	}

	if end-start > maxBlockRange {
		return nil, fmt.Errorf("max allowed block range (%d) exceeded", maxBlockRange)
	}

	eventLogs, err := GetBlockLogRange(blockStore, state, start, end, ethFilter.EthBlockFilter, readReceipts, evmAuxStore)
	if err != nil {
		return nil, err
	}

	return proto.Marshal(&ptypes.EthFilterLogList{EthBlockLogs: eventLogs})
}

func GetBlockLogRange(
	blockStore store.BlockStore,
	state loomchain.ReadOnlyState,
	from, to uint64,
	ethFilter eth.EthBlockFilter,
	readReceipts loomchain.ReadReceiptHandler,
	evmAuxStore *evmaux.EvmAuxStore,
) ([]*ptypes.EthFilterLog, error) {
	if from > to {
		return nil, fmt.Errorf("from block (%v) greater than to block (%v)", from, to)
	}
	eventLogs := []*ptypes.EthFilterLog{}

	for height := from; height <= to; height++ {
		blockLogs, err := getBlockLogs(blockStore, state, ethFilter, height, readReceipts, evmAuxStore)
		if err != nil {
			return nil, err
		}
		eventLogs = append(eventLogs, blockLogs...)
	}
	return eventLogs, nil
}

func getBlockLogs(
	blockStore store.BlockStore,
	state loomchain.ReadOnlyState,
	ethFilter eth.EthBlockFilter,
	height uint64,
	readReceipts loomchain.ReadReceiptHandler,
	evmAuxStore *evmaux.EvmAuxStore,
) ([]*ptypes.EthFilterLog, error) {

	bloomFilter := evmAuxStore.GetBloomFilter(height)
	if len(bloomFilter) > 0 {
		if MatchBloomFilter(ethFilter, bloomFilter) {
			txObject, err := GetBlockByNumber(blockStore, state, int64(height), false, evmAuxStore)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get block at height %d", height)
			}

			var logsBlock []*ptypes.EthFilterLog
			for _, txHashData := range txObject.Transactions {
				txHash, err := eth.DecDataToBytes(txHashData.(eth.Data))
				if err != nil {
					return nil, errors.Wrapf(err, "unable to decode txhash %x", txHashData)
				}

				txReceipt, err := readReceipts.GetReceipt(txHash)
				if errors.Cause(err) == common.ErrTxReceiptNotFound {
					continue
				} else if err != nil {
					return nil, errors.Wrap(err, "failed to load receipt")
				}
				logsTx, err := getTxHashLogs(blockStore, txReceipt, ethFilter, txHash)
				if err != nil {
					return nil, errors.Wrap(err, "failed to load tx logs")
				}
				logsBlock = append(logsBlock, logsTx...)
			}
			return logsBlock, nil
		}
	}
	return nil, nil
}

func GetPendingBlockLogs(
	blockStore store.BlockStore, ethFilter eth.EthBlockFilter, receiptHandler loomchain.ReadReceiptHandler,
) ([]*ptypes.EthFilterLog, error) {
	txHashList := receiptHandler.GetPendingTxHashList()
	var logsBlock []*ptypes.EthFilterLog
	for _, txHash := range txHashList {
		txReceipt, err := receiptHandler.GetPendingReceipt(txHash)
		if err != nil {
			return nil, errors.Wrap(err, "cannot find pending tx receipt matching hash")
		}
		logsTx, err := getTxHashLogs(blockStore, txReceipt, ethFilter, txHash)
		if err != nil {
			return nil, errors.Wrap(err, "logs for tx")
		}
		logsBlock = append(logsBlock, logsTx...)
	}
	return logsBlock, nil
}

func getTxHashLogs(
	blockStore store.BlockStore,
	txReceipt ptypes.EvmTxReceipt,
	filter eth.EthBlockFilter,
	txHash []byte,
) ([]*ptypes.EthFilterLog, error) {
	var blockLogs []*ptypes.EthFilterLog

	// Timestamp added here rather than being stored in the event itself so
	// as to avoid altering the data saved to the app-store.
	var timestamp int64
	if len(txReceipt.Logs) > 0 {
		height := int64(txReceipt.BlockNumber)
		var blockResult *ctypes.ResultBlock
		blockResult, err := blockStore.GetBlockByHeight(&height)
		if err != nil {
			return blockLogs, errors.Wrapf(err, "getting block info for height %v", height)
		}
		timestamp = blockResult.Block.Header.Time.Unix()
	}

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
				BlockTime:        timestamp,
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
