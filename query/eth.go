// +build evm

package query

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"strconv"
)

const (
	FilterMaxOrTopics = 2
	SolidtyMaxTopics  = 4
)

func QueryChain(query string, state loomchain.ReadOnlyState) ([]byte, error) {
	ethFilter, err := unmarshalEthFilter([]byte(query))
	if err != nil {
		return nil, err
	}
	start, err := blockNumber(ethFilter.FromBlock, uint64(state.Block().Height))
	if err != nil {
		return nil, err
	}
	end, err := blockNumber(ethFilter.ToBlock, uint64(state.Block().Height))
	if err != nil {
		return nil, err
	}
	if start > end {
		return nil, fmt.Errorf("to block before end block")
	}

	eventLogs := []*types.EthFilterLog{}

	for height := start; height <= end; height++ {
		blockLogs, err := GetBlockLogs(ethFilter.EthBlockFilter, state, height)
		if err != nil {
			return nil, err
		}
		eventLogs = append(eventLogs, blockLogs...)
	}

	return proto.Marshal(&types.EthFilterLogList{eventLogs})
}

func GetBlockLogs(ethFilter EthBlockFilter, state loomchain.ReadOnlyState, height uint64) ([]*types.EthFilterLog, error) {
	heightB := BlockHeightToBytes(height)
	bloomState := store.PrefixKVReader(BloomPrefix, state)
	bloomFilter := bloomState.Get(heightB)
	if len(bloomFilter) > 0 {
		if matchBloomFilter(ethFilter, bloomFilter) {
			txHashState := store.PrefixKVReader(TxHashPrefix, state)
			txHash := txHashState.Get(heightB)
			return getTxHashLogs(state, ethFilter, txHash)
		}
	}
	return nil, nil
}

func getTxHashLogs(state loomchain.ReadOnlyState, filter EthBlockFilter, txHash []byte) ([]*types.EthFilterLog, error) {
	receiptState := store.PrefixKVReader(ReceiptPrefix, state)
	txReceiptProto := receiptState.Get(txHash)
	txReceipt := types.EvmTxReceipt{}
	err := proto.Unmarshal(txReceiptProto, &txReceipt)
	if err != nil {
		return nil, err
	}
	var blockLogs []*types.EthFilterLog

	for i, eventLog := range txReceipt.Logs {
		if matchEthFilter(filter, *eventLog) {
			var topics [][]byte
			for _, topic := range eventLog.Topics {
				topics = append(topics, []byte(topic))
			}
			blockLogs = append(blockLogs, &types.EthFilterLog{
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

func blockNumber(bockTag string, height uint64) (uint64, error) {
	var block uint64
	switch bockTag {
	case "":
		block = height - 1
	case "latest":
		block = height - 1
	case "pending":
		block = height
	case "earliest":
		return uint64(1), nil
	default:
		var err error
		block, err = strconv.ParseUint(bockTag, 0, 64)
		if err != nil {
			return block, err
		}
	}
	if block < 1 {
		block = 1
	}
	return block, nil
}

func unmarshalEthFilter(query []byte) (EthFilter, error) {
	var filter struct {
		FromBlock string        `json:"fromBlock"`
		ToBlock   string        `json:"toBlock"`
		Address   string        `json:"address"`
		Topics    []interface{} `json:"topics"`
	}
	json.Unmarshal(query, &filter)

	rFilter := EthFilter{
		FromBlock: filter.FromBlock,
		ToBlock:   filter.ToBlock,
	}

	if len(filter.Address) > 0 {
		address, err := loom.LocalAddressFromHexString(filter.Address)
		if err != nil {
			return EthFilter{}, fmt.Errorf("invalid ethfilter, address")
		}
		rFilter.Addresses = append(rFilter.Addresses, address)
	}

	if len(filter.Topics) > SolidtyMaxTopics {
		return EthFilter{}, fmt.Errorf("invalid ethfilter, too many topics")
	}
	for _, topicUT := range filter.Topics {
		switch topic := topicUT.(type) {
		case string:
			rFilter.Topics = append(rFilter.Topics, []string{topic})
		case nil:
			rFilter.Topics = append(rFilter.Topics, nil)
		case []interface{}:
			topicPairUT := topicUT.([]interface{})
			if len(topicPairUT) != FilterMaxOrTopics {
				return EthFilter{}, fmt.Errorf("invalid ethfilter, can only OR two topics")
			}
			var topic1, topic2 string
			switch topic := topicPairUT[0].(type) {
			case string:
				topic1 = string(topic)
			default:
				return EthFilter{}, fmt.Errorf("invalid ethfilter, unreconised topic pair")
			}
			switch topic := topicPairUT[1].(type) {
			case string:
				topic2 = string(topic)
			default:
				return EthFilter{}, fmt.Errorf("invalid ethfilter, unreconised topic pair")
			}
			rFilter.Topics = append(rFilter.Topics, []string{topic1, topic2})
		default:
			return EthFilter{}, fmt.Errorf("invalid ethfilter, unrecognised topic")
		}
	}

	return rFilter, nil
}

func matchBloomFilter(ethFilter EthBlockFilter, bloomFilter []byte) bool {
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

func matchEthFilter(filter EthBlockFilter, eventLog types.EventData) bool {
	if len(filter.Topics) > len(eventLog.Topics) {
		return false
	}

	if len(filter.Addresses) > 0 {
		found := false
		for _, addr := range filter.Addresses {
			if 0 == addr.Compare(eventLog.Address.Local) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	for i, topics := range filter.Topics {
		if topics != nil {
			found := false
			for _, topic := range topics {
				if topic == eventLog.Topics[i] {
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

func BlockHeightToBytes(height uint64) []byte {
	heightB := make([]byte, 8)
	binary.LittleEndian.PutUint64(heightB, height)
	return heightB
}
