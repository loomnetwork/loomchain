package query

import (
	"bytes"
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	types1 "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/abci/types"
	"testing"
)

const (
	testFilter  = "{\"fromBlock\":\"0x1\",\"toBlock\":\"0x2\",\"address\":\"0x8888f1f195afa192cfee860698584c030f4c9db1\",\"topics\":[\"0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b\",null,[\"0x000000000000000000000000a94f5374fce5edbc8e2a8697c15331677e6ebf0b\",\"0x0000000000000000000000000aff3454fce5edbc8cca8697c15331677e6ebccc\"]]}"
	emptyFilter = "{\"fromBlock\":\"0x0\",\"toBlock\":\"latest\",\"address\":\"\",\"topics\":[]}"
)

func TestEthUnmarshal(t *testing.T) {
	_, err := unmarshalEthFilter([]byte(testFilter))
	require.NoError(t, err, "un-marshalling test filter")

	_, err = unmarshalEthFilter([]byte(emptyFilter))
	require.NoError(t, err, "un-marshalling test filter")
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
	ethFilter1 := EthBlockFilter{
		Topics: [][]string{{"Topic1"}, nil, {"Topic3", "Topic4"}, {"Topic4"}},
	}
	ethFilter2 := EthBlockFilter{
		Addresses: []loom.LocalAddress{addr2.Local},
	}
	ethFilter3 := EthBlockFilter{
		Topics: [][]string{{"Topic1"}},
	}
	ethFilter4 := EthBlockFilter{
		Addresses: []loom.LocalAddress{addr2.Local, addr1.Local},
		Topics:    [][]string{nil, nil, {"Topic2"}},
	}
	bloomFilter := GenBloomFilter(testEvents)

	require.True(t, matchBloomFilter(ethFilter1, bloomFilter))
	require.False(t, matchBloomFilter(ethFilter2, bloomFilter))
	require.True(t, matchBloomFilter(ethFilter3, bloomFilter))
	require.False(t, matchBloomFilter(ethFilter4, bloomFilter))

	require.True(t, matchEthFilter(ethFilter1, *testEventsG[0]))
	require.False(t, matchEthFilter(ethFilter2, *testEventsG[0]))
	require.True(t, matchEthFilter(ethFilter3, *testEventsG[0]))
	require.False(t, matchEthFilter(ethFilter4, *testEventsG[0]))

	require.False(t, matchEthFilter(ethFilter1, *testEventsG[1]))
	require.False(t, matchEthFilter(ethFilter2, *testEventsG[1]))
	require.False(t, matchEthFilter(ethFilter3, *testEventsG[1]))
	require.False(t, matchEthFilter(ethFilter4, *testEventsG[1]))
}

func TestGetLogs(t *testing.T) {
	addr1 := &types1.Address{
		ChainId: "defult",
		Local:   []byte("testLocal1"),
	}
	ethFilter := EthBlockFilter{
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
	state := mockState()
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

	receiptState := store.PrefixKVStore(ReceiptPrefix, state)
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

func TestBlockNumber(t *testing.T) {
	var height = uint64(50)

	block, err := blockNumber("23", height)
	require.NoError(t, err)
	require.Equal(t, block, uint64(23))

	block, err = blockNumber("0x17", height)
	require.NoError(t, err)
	require.Equal(t, block, uint64(23))

	block, err = blockNumber("latest", height)
	require.NoError(t, err)
	require.Equal(t, block, height)

	block, err = blockNumber("earliest", height)
	require.Error(t, err)

	_, err = blockNumber("pending", height)
	require.Error(t, err)

	_, err = blockNumber("nonsense", height)
	require.Error(t, err)
}

func mockState() loomchain.State {
	return loomchain.NewStoreState(context.Background(), store.NewMemStore(), abci.Header{})
}
