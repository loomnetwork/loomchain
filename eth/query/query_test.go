// +build evm

package query

import (
	"bytes"
	"os"
	"testing"

	"github.com/loomnetwork/loomchain/events"

	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/leveldb"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	types1 "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/bloom"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/receipts/handler"
	"github.com/stretchr/testify/require"
)

const (
	allFilter = "{\"fromBlock\":\"0x0\",\"toBlock\":\"latest\",\"address\":\"\",\"topics\":[]}"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestQueryChain(t *testing.T) {
	testQueryChain(t, handler.ReceiptHandlerChain)
	os.RemoveAll(leveldb.Db_Filename)
	_, err := os.Stat(leveldb.Db_Filename)
	require.True(t, os.IsNotExist(err))
	testQueryChain(t, handler.ReceiptHandlerLevelDb)
}

func testQueryChain(t *testing.T, v handler.ReceiptHandlerVersion) {
	eventDispatcher := events.NewLogEventDispatcher()
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)
	receiptHandler, err := handler.NewReceiptHandler(v, eventHandler, leveldb.Default_DBHeight)
	var writer loomchain.WriteReceiptHandler
	writer = receiptHandler

	require.NoError(t, err)
	state := common.MockState(0)

	state4 := common.MockStateAt(state, 4)
	mockEvent1 := []*loomchain.EventData{
		{
			Topics:      []string{"topic1", "topic2", "topic3"},
			EncodedBody: []byte("somedata"),
			Address:     addr1.MarshalPB(),
		},
	}
	_, err = writer.CacheReceipt(state4, addr1, addr2, mockEvent1, nil)
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()

	protoBlock, err := GetPendingBlock(4, true, receiptHandler)
	require.NoError(t, err)
	blockInfo := types.EthBlockInfo{}
	require.NoError(t, proto.Unmarshal(protoBlock, &blockInfo))
	require.EqualValues(t, int64(4), blockInfo.Number)
	require.EqualValues(t, 1, len(blockInfo.Transactions))

	require.NoError(t, receiptHandler.CommitBlock(state4, 4))

	mockEvent2 := []*loomchain.EventData{
		{
			Topics:      []string{"topic1"},
			EncodedBody: []byte("somedata"),
			Address:     addr1.MarshalPB(),
		},
	}
	state20 := common.MockStateAt(state, 20)
	_, err = writer.CacheReceipt(state20, addr1, addr2, mockEvent2, nil)
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(state20, 20))

	state30 := common.MockStateAt(state, uint64(30))
	result, err := QueryChain(allFilter, state30, receiptHandler)
	require.NoError(t, err, "error query chain, filter is %s", allFilter)
	var logs types.EthFilterLogList
	require.NoError(t, proto.Unmarshal(result, &logs), "unmarshalling EthFilterLogList")
	require.Equal(t, 2, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.NoError(t, receiptHandler.Close())
}

func TestMatchFilters(t *testing.T) {
	addr1 := &types1.Address{
		ChainId: "defult",
		Local:   []byte("testLocal1"),
	}
	addr2 := &types1.Address{
		ChainId: "defult",
		Local:   []byte("testLocal2"),
	}
	testEvents := []*loomchain.EventData{
		{
			Topics:  []string{"Topic1", "Topic2", "Topic3", "Topic4"},
			Address: addr1,
		},
		{
			Topics:  []string{"Topic5"},
			Address: addr1,
		},
	}
	testEventsG := []*types.EventData{
		{
			Topics:      []string{"Topic1", "Topic2", "Topic3", "Topic4"},
			Address:     addr1,
			EncodedBody: []byte("Some data"),
		},
		{
			Topics:  []string{"Topic5"},
			Address: addr1,
		},
	}
	ethFilter1 := utils.EthBlockFilter{
		Topics: [][]string{{"Topic1"}, nil, {"Topic3", "Topic4"}, {"Topic4"}},
	}
	ethFilter2 := utils.EthBlockFilter{
		Addresses: []loom.LocalAddress{addr2.Local},
	}
	ethFilter3 := utils.EthBlockFilter{
		Topics: [][]string{{"Topic1"}},
	}
	ethFilter4 := utils.EthBlockFilter{
		Addresses: []loom.LocalAddress{addr2.Local, addr1.Local},
		Topics:    [][]string{nil, nil, {"Topic2"}},
	}
	ethFilter5 := utils.EthBlockFilter{
		Topics: [][]string{{"Topic1"}, {"Topic6"}},
	}
	bloomFilter := bloom.GenBloomFilter(common.ConvertEventData(testEvents))

	require.True(t, MatchBloomFilter(ethFilter1, bloomFilter))
	require.False(t, MatchBloomFilter(ethFilter2, bloomFilter))
	require.True(t, MatchBloomFilter(ethFilter3, bloomFilter))
	require.False(t, MatchBloomFilter(ethFilter4, bloomFilter))
	require.False(t, MatchBloomFilter(ethFilter5, bloomFilter))

	require.True(t, utils.MatchEthFilter(ethFilter1, *testEventsG[0]))
	require.False(t, utils.MatchEthFilter(ethFilter2, *testEventsG[0]))
	require.True(t, utils.MatchEthFilter(ethFilter3, *testEventsG[0]))
	require.False(t, utils.MatchEthFilter(ethFilter4, *testEventsG[0]))
	require.False(t, utils.MatchEthFilter(ethFilter5, *testEventsG[0]))

	require.False(t, utils.MatchEthFilter(ethFilter1, *testEventsG[1]))
	require.False(t, utils.MatchEthFilter(ethFilter2, *testEventsG[1]))
	require.False(t, utils.MatchEthFilter(ethFilter3, *testEventsG[1]))
	require.False(t, utils.MatchEthFilter(ethFilter4, *testEventsG[1]))
	require.False(t, utils.MatchEthFilter(ethFilter5, *testEventsG[1]))
}

func TestGetLogs(t *testing.T) {
	testGetLogs(t, handler.ReceiptHandlerChain)

	os.RemoveAll(leveldb.Db_Filename)
	_, err := os.Stat(leveldb.Db_Filename)
	require.True(t, os.IsNotExist(err))
	testGetLogs(t, handler.ReceiptHandlerLevelDb)
}

func testGetLogs(t *testing.T, v handler.ReceiptHandlerVersion) {
	os.RemoveAll(leveldb.Db_Filename)
	_, err := os.Stat(leveldb.Db_Filename)
	require.True(t, os.IsNotExist(err))

	eventDispatcher := events.NewLogEventDispatcher()
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)
	receiptHandler, err := handler.NewReceiptHandler(v, eventHandler, leveldb.Default_DBHeight)
	var writer loomchain.WriteReceiptHandler
	writer = receiptHandler

	require.NoError(t, err)
	ethFilter := utils.EthBlockFilter{
		Topics: [][]string{{"Topic1"}, nil, {"Topic3", "Topic4"}, {"Topic4"}},
	}
	testEvents := []*loomchain.EventData{
		{
			Topics:      []string{"Topic1", "Topic2", "Topic3", "Topic4"},
			Address:     addr1.MarshalPB(),
			EncodedBody: []byte("Some data"),
		},
		{
			Topics:  []string{"Topic5"},
			Address: addr1.MarshalPB(),
		},
	}

	testEventsG := []*loomchain.EventData{
		{
			Topics:      []string{"Topic1", "Topic2", "Topic3", "Topic4"},
			Address:     addr1.MarshalPB(),
			EncodedBody: []byte("Some data"),
		},
		{
			Topics:  []string{"Topic5"},
			Address: addr1.MarshalPB(),
		},
	}
	state := common.MockState(1)
	state32 := common.MockStateAt(state, 32)
	txHash, err := writer.CacheReceipt(state32, addr1, addr2, testEventsG, nil)
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(state32, 32))

	state40 := common.MockStateAt(state, 40)
	txReceipt, err := receiptHandler.GetReceipt(state40, txHash)
	require.NoError(t, err)
	logs, err := getTxHashLogs(txReceipt, ethFilter, txHash)
	require.NoError(t, err, "getBlockLogs failed")
	require.Equal(t, len(logs), 1)
	require.Equal(t, logs[0].TransactionIndex, txReceipt.TransactionIndex)
	require.Equal(t, logs[0].TransactionHash, txReceipt.TxHash)
	require.True(t, 0 == bytes.Compare(logs[0].BlockHash, txReceipt.BlockHash))
	require.Equal(t, logs[0].BlockNumber, txReceipt.BlockNumber)
	require.True(t, 0 == bytes.Compare(logs[0].Address, txReceipt.CallerAddress.Local))
	require.True(t, 0 == bytes.Compare(logs[0].Data, testEvents[0].EncodedBody))
	require.Equal(t, len(logs[0].Topics), 4)
	require.True(t, 0 == bytes.Compare(logs[0].Topics[0], []byte(testEvents[0].Topics[0])))

	require.NoError(t, receiptHandler.Close())
}
