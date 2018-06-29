// +build evm

package query

import (
	"bytes"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	types1 "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/utils"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	allFilter = "{\"fromBlock\":\"0x0\",\"toBlock\":\"latest\",\"address\":\"\",\"topics\":[]}"
)

func TestQueryChain(t *testing.T) {
	contract, err := loom.LocalAddressFromHexString("0x1234567890123456789012345678901234567890")
	require.NoError(t, err)
	receipts := []MockReceipt{
		{
			Height:   uint64(4),
			Contract: contract,
			Events: []MockEvent{
				{
					Topics: []string{"topic1", "topic2", "topic3"},
					Data:   []byte("somedata"),
				},
			},
		},
		{
			Height:   uint64(20),
			Contract: contract,
			Events: []MockEvent{
				{
					Topics: []string{"topic1"},
					Data:   []byte("somedata2"),
				},
			},
		},
	}
	state, err := MockPopulatedState(receipts)
	require.NoError(t, err, "setting up mock state")
	state = MockStateAt(state, int64(30))
	result, err := QueryChain(allFilter, state)
	require.NoError(t, err, "error query chain, filter is %s", allFilter)
	var logs ptypes.EthFilterLogList
	require.NoError(t, proto.Unmarshal(result, &logs), "unmarshalling EthFilterLogList")
	require.Equal(t, 2, len(logs.EthBlockLogs), "wrong number of logs returned")
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
	bloomFilter := GenBloomFilter(testEvents)

	require.True(t, MatchBloomFilter(ethFilter1, bloomFilter))
	require.False(t, MatchBloomFilter(ethFilter2, bloomFilter))
	require.True(t, MatchBloomFilter(ethFilter3, bloomFilter))
	require.False(t, MatchBloomFilter(ethFilter4, bloomFilter))
	require.False(t, MatchBloomFilter(ethFilter5, bloomFilter))

	require.True(t, MatchEthFilter(ethFilter1, *testEventsG[0]))
	require.False(t, MatchEthFilter(ethFilter2, *testEventsG[0]))
	require.True(t, MatchEthFilter(ethFilter3, *testEventsG[0]))
	require.False(t, MatchEthFilter(ethFilter4, *testEventsG[0]))
	require.False(t, MatchEthFilter(ethFilter5, *testEventsG[0]))

	require.False(t, MatchEthFilter(ethFilter1, *testEventsG[1]))
	require.False(t, MatchEthFilter(ethFilter2, *testEventsG[1]))
	require.False(t, MatchEthFilter(ethFilter3, *testEventsG[1]))
	require.False(t, MatchEthFilter(ethFilter4, *testEventsG[1]))
	require.False(t, MatchEthFilter(ethFilter5, *testEventsG[1]))
}

func TestGetLogs(t *testing.T) {
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
	txHash := []byte("MyHash")
	state := MockState()
	testReciept := types.EvmTxReceipt{
		TransactionIndex:  0,
		BlockHash:         []byte{},
		BlockNumber:       32,
		CumulativeGasUsed: 0,
		GasUsed:           0,
		ContractAddress:   addr1.Local,
		Logs:              testEventsG,
		LogsBloom:         GenBloomFilter(testEvents),
		Status:            1,
	}

	protoTestReceipt, err := proto.Marshal(&testReciept)
	require.NoError(t, err, "marshaling")

	receiptState := store.PrefixKVStore(utils.ReceiptPrefix, state)
	receiptState.Set(txHash, protoTestReceipt)

	logs, err := getTxHashLogs(state, ethFilter, txHash)
	require.NoError(t, err, "getBlockLogs failed")
	require.Equal(t, len(logs), 1)
	require.Equal(t, logs[0].TransactionIndex, testReciept.TransactionIndex)
	require.Equal(t, logs[0].TransactionHash, txHash)
	require.True(t, 0 == bytes.Compare(logs[0].BlockHash, testReciept.BlockHash))
	require.Equal(t, logs[0].BlockNumber, testReciept.BlockNumber)
	require.True(t, 0 == bytes.Compare(logs[0].Address, testReciept.ContractAddress))
	require.True(t, 0 == bytes.Compare(logs[0].Data, testEvents[0].EncodedBody))
	require.Equal(t, len(logs[0].Topics), 4)
	require.True(t, 0 == bytes.Compare(logs[0].Topics[0], []byte(testEvents[0].Topics[0])))
}
