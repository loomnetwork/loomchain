// +build evm

package query

import (
	"bytes"
	"os"
	"testing"
	
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
)

func TestQueryChain(t *testing.T) {
	testQueryChain(t, handler.ReceiptHandlerChain)
	os.RemoveAll(leveldb.Db_Filename)
	_, err := os.Stat(leveldb.Db_Filename)
	require.True(t,os.IsNotExist(err))
	testQueryChain(t, handler.ReceiptHandlerLevelDb)
}

func testQueryChain(t *testing.T, v handler.ReceiptHandlerVersion) {
	receiptHandler, err := handler.NewReceiptHandler(v, &loomchain.DefaultEventHandler{})
	require.NoError(t, err)
	state:= common.MockState(0)
	
	mockEvent1 := []*types.EventData{
		{
			Topics:      []string{"topic1", "topic2", "topic3"},
			EncodedBody: []byte("somedata"),
			Address:     addr1.MarshalPB(),
		},
	}
	receipts4 := []*types.EvmTxReceipt{common.MakeDummyReceipt(t,4,0,mockEvent1)}
	receiptHandler.ReceiptsCache = receipts4
	state4 := common.MockStateAt(state, 4)
	receiptHandler.CommitBlock(state4, 4)
	
	mockEvent2 := []*types.EventData{
		{
			Topics: []string{"topic1"},
			EncodedBody:   []byte("somedata"),
			Address:     addr1.MarshalPB(),
		},
	}
	receipts20 := []*types.EvmTxReceipt{common.MakeDummyReceipt(t,20,0,mockEvent2)}
	receiptHandler.ReceiptsCache = receipts20
	state20 := common.MockStateAt(state, 20)
	receiptHandler.CommitBlock(state20, 20)
	
	state30 := MockStateAt(state, int64(30))
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
	bloomFilter := bloom.GenBloomFilter(ConvertEventData(testEvents))

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
	require.True(t,os.IsNotExist(err))
	testGetLogs(t, handler.ReceiptHandlerLevelDb)
}

func testGetLogs(t *testing.T, v handler.ReceiptHandlerVersion) {
	os.RemoveAll(leveldb.Db_Filename)
	_, err := os.Stat(leveldb.Db_Filename)
	require.True(t,os.IsNotExist(err))
	receiptHandler, err := handler.NewReceiptHandler(v, &loomchain.DefaultEventHandler{})
	require.NoError(t, err)
	addr1 := &types1.Address{
		ChainId: "defult",
		Local:   []byte("testLocal1"),
	}
	ethFilter := utils.EthBlockFilter{
		Topics: [][]string{{"Topic1"}, nil, {"Topic3", "Topic4"}, {"Topic4"}},
	}
	testEvents := []*loomchain.EventData{
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
	state := common.MockState(1)
	testReceipts := []*types.EvmTxReceipt{common.MakeDummyReceipt(t,32,0,testEventsG)}
	testReceipts[0].ContractAddress = addr1.Local
	receiptHandler.ReceiptsCache = testReceipts
	state32 := common.MockStateAt(state, 32)
	receiptHandler.CommitBlock(state32, 32)
	
	state40 := common.MockStateAt(state, 40)
	logs, err := getTxHashLogs(state40, receiptHandler, ethFilter, testReceipts[0].TxHash)
	require.NoError(t, err, "getBlockLogs failed")
	require.Equal(t, len(logs), 1)
	require.Equal(t, logs[0].TransactionIndex, testReceipts[0].TransactionIndex)
	require.Equal(t, logs[0].TransactionHash, testReceipts[0].TxHash)
	require.True(t, 0 == bytes.Compare(logs[0].BlockHash, testReceipts[0].BlockHash))
	require.Equal(t, logs[0].BlockNumber, testReceipts[0].BlockNumber)
	require.True(t, 0 == bytes.Compare(logs[0].Address, testReceipts[0].ContractAddress))
	require.True(t, 0 == bytes.Compare(logs[0].Data, testEvents[0].EncodedBody))
	require.Equal(t, len(logs[0].Topics), 4)
	require.True(t, 0 == bytes.Compare(logs[0].Topics[0], []byte(testEvents[0].Topics[0])))
	
	require.NoError(t, receiptHandler.Close())
}

func ConvertEventData(events []*loomchain.EventData) []*types.EventData {
	var typesEvents []*types.EventData
	for _, event := range events {
		typeEvent := types.EventData(*event)
		typesEvents = append(typesEvents, &typeEvent)
	}
	return typesEvents
}



