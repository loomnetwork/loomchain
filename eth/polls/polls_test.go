// +build evm

package polls

import (
	"strconv"
	"sync"
	"testing"

	"github.com/loomnetwork/loomchain/rpc/eth"

	"github.com/loomnetwork/loomchain/events"
	"github.com/loomnetwork/loomchain/store"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/handler"
	"github.com/stretchr/testify/require"
)

var (
	addr1    = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	contract = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestLogPoll(t *testing.T) {
	testLogPoll(t)
}

func testLogPoll(t *testing.T) {
	evmAuxStore, err := common.NewMockEvmAuxStore()
	require.NoError(t, err)
	blockStore := store.NewMockBlockStore()
	eventDispatcher := events.NewLogEventDispatcher()
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)
	receiptHandler := handler.NewReceiptHandler(eventHandler, handler.DefaultMaxReceipts, evmAuxStore)
	sub := NewEthSubscriptions(evmAuxStore, blockStore)
	allFilter := eth.JsonFilter{
		FromBlock: "earliest",
		ToBlock:   "pending",
		Address:   nil,
		Topics:    nil,
	}
	state := makeMockState(t, receiptHandler)
	ethFilter, err := eth.DecLogFilter(allFilter)
	require.NoError(t, err)
	id, err := sub.AddLogPoll(ethFilter, 1)
	require.NoError(t, err)

	state5 := common.MockStateAt(state, uint64(5))
	result, err := sub.LegacyPoll(state5, id, receiptHandler)
	require.NoError(t, err)
	var envolope types.EthFilterEnvelope
	var logs *types.EthFilterLogList
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 1, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.Equal(t, "height4", string(logs.EthBlockLogs[0].Data))
	state40 := common.MockStateAt(state, uint64(40))
	result, err = sub.LegacyPoll(state40, id, receiptHandler)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 3, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.Equal(t, "height20", string(logs.EthBlockLogs[0].Data))
	require.Equal(t, "height25", string(logs.EthBlockLogs[1].Data))
	require.Equal(t, "height30", string(logs.EthBlockLogs[2].Data))

	state50 := common.MockStateAt(state, uint64(50))
	result, err = sub.LegacyPoll(state50, id, receiptHandler)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 0, len(logs.EthBlockLogs), "wrong number of logs returned")
	state60 := common.MockStateAt(state, uint64(60))
	sub.Remove(id)
	_, err = sub.LegacyPoll(state60, id, receiptHandler)
	require.Error(t, err, "subscription not removed")
	require.NoError(t, receiptHandler.Close())
	evmAuxStore.ClearData()
}

func TestTxPoll(t *testing.T) {
	testLegacyTxPoll(t)
	testTxPoll(t)
}

func testLegacyTxPoll(t *testing.T) {
	evmAuxStore, err := common.NewMockEvmAuxStore()
	require.NoError(t, err)
	blockStore := store.NewMockBlockStore()
	eventDispatcher := events.NewLogEventDispatcher()
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)
	receiptHandler := handler.NewReceiptHandler(eventHandler, handler.DefaultMaxReceipts, evmAuxStore)

	sub := NewEthSubscriptions(evmAuxStore, blockStore)
	state := makeMockState(t, receiptHandler)
	id := sub.AddTxPoll(uint64(5))

	var envolope types.EthFilterEnvelope
	var txHashes *types.EthTxHashList
	state27 := common.MockStateAt(state, uint64(27))
	result, err := sub.LegacyPoll(state27, id, receiptHandler)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 2, len(txHashes.EthTxHash), "wrong number of logs returned")

	state50 := common.MockStateAt(state, uint64(50))
	result, err = sub.LegacyPoll(state50, id, receiptHandler)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 1, len(txHashes.EthTxHash), "wrong number of logs returned")

	state60 := common.MockStateAt(state, uint64(60))
	sub.Remove(id)
	_, err = sub.LegacyPoll(state60, id, receiptHandler)
	require.Error(t, err, "subscription not removed")
	require.NoError(t, receiptHandler.Close())
}

func testTxPoll(t *testing.T) {
	evmAuxStore, err := common.NewMockEvmAuxStore()
	require.NoError(t, err)
	blockStore := store.NewMockBlockStore()
	eventDispatcher := events.NewLogEventDispatcher()
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)
	receiptHandler := handler.NewReceiptHandler(eventHandler, handler.DefaultMaxReceipts, evmAuxStore)

	sub := NewEthSubscriptions(evmAuxStore, blockStore)
	state := makeMockState(t, receiptHandler)
	id := sub.AddTxPoll(uint64(5))

	state27 := common.MockStateAt(state, uint64(27))
	result, err := sub.Poll(state27, id, receiptHandler)
	require.NoError(t, err)
	require.NotEqual(t, nil, result)
	data, ok := result.([]eth.Data)
	require.True(t, ok)
	require.Equal(t, 2, len(data), "wrong number of logs returned")

	state50 := common.MockStateAt(state, uint64(50))
	result, err = sub.Poll(state50, id, receiptHandler)
	require.NoError(t, err)
	require.NotEqual(t, nil, result)
	data, ok = result.([]eth.Data)
	require.True(t, ok)
	require.Equal(t, 1, len(data), "wrong number of logs returned")

	state105 := common.MockStateAt(state, uint64(105))
	result, err = sub.Poll(state105, id, receiptHandler)
	require.NoError(t, err)
	require.NotEqual(t, nil, result)
	data, ok = result.([]eth.Data)
	require.True(t, ok)
	require.Equal(t, 5, len(data), "wrong number of logs returned")

	state115 := common.MockStateAt(state, uint64(115))
	result, err = sub.Poll(state115, id, receiptHandler)
	require.NoError(t, err)
	require.NotEqual(t, nil, result)
	data, ok = result.([]eth.Data)
	require.True(t, ok)
	require.Equal(t, 10, len(data), "wrong number of logs returned")

	state140 := common.MockStateAt(state, uint64(140))
	result, err = sub.Poll(state140, id, receiptHandler)
	require.NoError(t, err)
	require.NotEqual(t, nil, result)
	data, ok = result.([]eth.Data)
	require.True(t, ok)
	require.Equal(t, 5, len(data), "wrong number of logs returned")

	state220 := common.MockStateAt(state, uint64(220))

	var wg sync.WaitGroup
	wg.Add(2)
	go func(s *EthSubscriptions) {
		defer wg.Done()
		result, err = s.Poll(state220, id, receiptHandler)
	}(sub)
	go func(s *EthSubscriptions) {
		defer wg.Done()
		s.Remove(id)
	}(sub)
	wg.Wait()

	result, err = sub.Poll(state220, id, receiptHandler)
	require.Error(t, err, "subscription not removed")
	require.NoError(t, receiptHandler.Close())
}

func TestTimeout(t *testing.T) {
	testTimeout(t, handler.ReceiptHandlerLevelDb)
}

func testTimeout(t *testing.T, version handler.ReceiptHandlerVersion) {
	evmAuxStore, err := common.NewMockEvmAuxStore()
	require.NoError(t, err)
	blockStore := store.NewMockBlockStore()
	eventDispatcher := events.NewLogEventDispatcher()
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)
	receiptHandler := handler.NewReceiptHandler(eventHandler, handler.DefaultMaxReceipts, evmAuxStore)

	BlockTimeout = 10
	sub := NewEthSubscriptions(evmAuxStore, blockStore)
	state := makeMockState(t, receiptHandler)

	var envolope types.EthFilterEnvelope
	var txHashes *types.EthTxHashList
	id := sub.AddTxPoll(uint64(1))

	state5 := common.MockStateAt(state, uint64(5))
	_ = sub.AddTxPoll(uint64(5))

	result, err := sub.LegacyPoll(state5, id, receiptHandler)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 1, len(txHashes.EthTxHash), "wrong number of logs returned")

	state12 := common.MockStateAt(state, uint64(12))
	_ = sub.AddTxPoll(uint64(12))

	result, err = sub.LegacyPoll(state12, id, receiptHandler)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 0, len(txHashes.EthTxHash), "wrong number of logs returned")

	state40 := common.MockStateAt(state, uint64(40))
	_ = sub.AddTxPoll(uint64(40))

	result, err = sub.LegacyPoll(state40, id, receiptHandler)
	require.Error(t, err, "poll did not timed out")
	require.NoError(t, receiptHandler.Close())
}

func makeMockState(t *testing.T, receiptHandler *handler.ReceiptHandler) loomchain.State {
	state := common.MockState(0)

	mockEvent4 := []*types.EventData{
		{
			Topics:      []string{"topic1", "topic2", "topic3"},
			EncodedBody: []byte("height4"),
			Address:     contract.MarshalPB(),
		},
	}
	state4 := common.MockStateAt(state, 4)

	_, err := receiptHandler.CacheReceipt(state4, addr1, contract, mockEvent4, nil, []byte{})
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(4))

	mockEvent20 := []*types.EventData{
		{
			Topics:      []string{"topic1"},
			EncodedBody: []byte("height20"),
			Address:     contract.MarshalPB(),
		},
	}
	state20 := common.MockStateAt(state, 20)
	_, err = receiptHandler.CacheReceipt(state20, addr1, contract, mockEvent20, nil, []byte{})
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(20))

	mockEvent25 := []*types.EventData{
		{
			Topics:      []string{"topic1"},
			EncodedBody: []byte("height25"),
			Address:     contract.MarshalPB(),
		},
	}
	state25 := common.MockStateAt(state, 25)
	_, err = receiptHandler.CacheReceipt(state25, addr1, contract, mockEvent25, nil, []byte{})
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(25))

	mockEvent30 := []*types.EventData{
		{
			Topics:      []string{"topic1", "topic2", "topic3"},
			EncodedBody: []byte("height30"),
			Address:     contract.MarshalPB(),
		},
	}
	state30 := common.MockStateAt(state, 30)
	_, err = receiptHandler.CacheReceipt(state30, addr1, contract, mockEvent30, nil, []byte{})
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(30))

	for height := 100; height < 120; height++ {
		mockEvent := []*types.EventData{
			{
				Topics:      []string{"topic1"},
				EncodedBody: []byte("height" + strconv.Itoa(height)),
				Address:     contract.MarshalPB(),
			},
		}
		state := common.MockStateAt(state, uint64(height))
		_, err = receiptHandler.CacheReceipt(state, addr1, contract, mockEvent, nil, []byte{})
		require.NoError(t, err)
		receiptHandler.CommitCurrentReceipt()
		require.NoError(t, receiptHandler.CommitBlock(int64(height)))
	}

	return state
}

func TestAddRemove(t *testing.T) {
	evmAuxStore, err := common.NewMockEvmAuxStore()
	require.NoError(t, err)
	blockStore := store.NewMockBlockStore()
	s := NewEthSubscriptions(evmAuxStore, blockStore)

	jsonFilter := eth.JsonFilter{
		FromBlock: "0x0",
		ToBlock:   "latest",
		Address:   nil,
		Topics:    nil,
	}
	myFilter, err := eth.DecLogFilter(jsonFilter)
	id, err := s.AddLogPoll(myFilter, 1)
	require.NoError(t, err)
	_, ok := s.polls[id]
	require.True(t, ok, "map key does not exists")

	s.Remove(id)
	_, ok = s.polls[id]
	require.False(t, ok, "id key not deleted")
}
