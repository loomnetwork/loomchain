// +build evm

package polls

import (
	"os"
	"testing"

	"github.com/loomnetwork/loomchain/events"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/receipts/handler"
	"github.com/loomnetwork/loomchain/receipts/leveldb"
	"github.com/stretchr/testify/require"
)

var (
	addr1    = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	contract = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestLogPoll(t *testing.T) {
	testLogPoll(t, handler.ReceiptHandlerChain)

	os.RemoveAll(leveldb.Db_Filename)
	_, err := os.Stat(leveldb.Db_Filename)
	require.True(t, os.IsNotExist(err))
	testLogPoll(t, handler.ReceiptHandlerLevelDb)
}

func testLogPoll(t *testing.T, version handler.ReceiptHandlerVersion) {
	eventDispatcher := events.NewLogEventDispatcher()
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)
	receiptHandler, err := handler.NewReceiptHandler(version, eventHandler, handler.DefaultMaxReceipts)
	require.NoError(t, err)

	sub := NewEthSubscriptions()
	allFilter := "{\"fromBlock\":\"0x0\",\"toBlock\":\"pending\",\"address\":\"\",\"topics\":[]}"
	state := makeMockState(t, receiptHandler)
	id, err := sub.AddLogPoll(allFilter, 1)
	require.NoError(t, err)

	state5 := common.MockStateAt(state, uint64(5))
	result, err := sub.Poll(state5, id, receiptHandler)
	require.NoError(t, err)

	var envolope types.EthFilterEnvelope
	var logs *types.EthFilterLogList
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 1, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.Equal(t, "height4", string(logs.EthBlockLogs[0].Data))

	state40 := common.MockStateAt(state, uint64(40))
	result, err = sub.Poll(state40, id, receiptHandler)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 3, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.Equal(t, "height20", string(logs.EthBlockLogs[0].Data))
	require.Equal(t, "height25", string(logs.EthBlockLogs[1].Data))
	require.Equal(t, "height30", string(logs.EthBlockLogs[2].Data))

	state50 := common.MockStateAt(state, uint64(50))
	result, err = sub.Poll(state50, id, receiptHandler)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 0, len(logs.EthBlockLogs), "wrong number of logs returned")

	state60 := common.MockStateAt(state, uint64(60))
	sub.Remove(id)
	result, err = sub.Poll(state60, id, receiptHandler)
	require.Error(t, err, "subscription not removed")
	receiptHandler.Close()
}

func TestTxPoll(t *testing.T) {
	testTxPoll(t, handler.ReceiptHandlerChain)

	os.RemoveAll(leveldb.Db_Filename)
	_, err := os.Stat(leveldb.Db_Filename)
	require.True(t, os.IsNotExist(err))
	testTxPoll(t, handler.ReceiptHandlerLevelDb)
}

func testTxPoll(t *testing.T, version handler.ReceiptHandlerVersion) {
	eventDispatcher := events.NewLogEventDispatcher()
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)
	receiptHandler, err := handler.NewReceiptHandler(version, eventHandler, handler.DefaultMaxReceipts)
	require.NoError(t, err)

	sub := NewEthSubscriptions()
	state := makeMockState(t, receiptHandler)
	id := sub.AddTxPoll(uint64(5))

	var envolope types.EthFilterEnvelope
	var txHashes *types.EthTxHashList
	state27 := common.MockStateAt(state, uint64(27))
	result, err := sub.Poll(state27, id, receiptHandler)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 2, len(txHashes.EthTxHash), "wrong number of logs returned")

	state50 := common.MockStateAt(state, uint64(50))
	result, err = sub.Poll(state50, id, receiptHandler)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 1, len(txHashes.EthTxHash), "wrong number of logs returned")

	state60 := common.MockStateAt(state, uint64(60))
	sub.Remove(id)
	result, err = sub.Poll(state60, id, receiptHandler)
	require.Error(t, err, "subscription not removed")
	receiptHandler.Close()
}

func TestTimeout(t *testing.T) {
	testTimeout(t, handler.ReceiptHandlerChain)

	os.RemoveAll(leveldb.Db_Filename)
	_, err := os.Stat(leveldb.Db_Filename)
	require.True(t, os.IsNotExist(err))
	testTimeout(t, handler.ReceiptHandlerLevelDb)
}

func testTimeout(t *testing.T, version handler.ReceiptHandlerVersion) {
	eventDispatcher := events.NewLogEventDispatcher()
	eventHandler := loomchain.NewDefaultEventHandler(eventDispatcher)
	receiptHandler, err := handler.NewReceiptHandler(version, eventHandler, handler.DefaultMaxReceipts)

	require.NoError(t, err)

	BlockTimeout = 10
	sub := NewEthSubscriptions()
	state := makeMockState(t, receiptHandler)

	var envolope types.EthFilterEnvelope
	var txHashes *types.EthTxHashList
	id := sub.AddTxPoll(uint64(1))

	state5 := common.MockStateAt(state, uint64(5))
	_ = sub.AddTxPoll(uint64(5))

	result, err := sub.Poll(state5, id, receiptHandler)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 1, len(txHashes.EthTxHash), "wrong number of logs returned")

	state12 := common.MockStateAt(state, uint64(12))
	_ = sub.AddTxPoll(uint64(12))

	result, err = sub.Poll(state12, id, receiptHandler)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 0, len(txHashes.EthTxHash), "wrong number of logs returned")

	state40 := common.MockStateAt(state, uint64(40))
	_ = sub.AddTxPoll(uint64(40))

	result, err = sub.Poll(state40, id, receiptHandler)
	require.Error(t, err, "poll did not timed out")
	receiptHandler.Close()
}

func makeMockState(t *testing.T, receiptHandler *handler.ReceiptHandler) loomchain.State {
	state := common.MockState(0)

	mockEvent4 := []*loomchain.EventData{
		{
			Topics:      []string{"topic1", "topic2", "topic3"},
			EncodedBody: []byte("height4"),
			Address:     contract.MarshalPB(),
		},
	}
	state4 := common.MockStateAt(state, 4)
	_, err := receiptHandler.CacheReceipt(state4, addr1, contract, mockEvent4, nil)
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(state4, 4))

	mockEvent20 := []*loomchain.EventData{
		{
			Topics:      []string{"topic1"},
			EncodedBody: []byte("height20"),
			Address:     contract.MarshalPB(),
		},
	}
	state20 := common.MockStateAt(state, 20)
	_, err = receiptHandler.CacheReceipt(state20, addr1, contract, mockEvent20, nil)
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(state20, 20))

	mockEvent25 := []*loomchain.EventData{
		{
			Topics:      []string{"topic1"},
			EncodedBody: []byte("height25"),
			Address:     contract.MarshalPB(),
		},
	}
	state25 := common.MockStateAt(state, 25)
	_, err = receiptHandler.CacheReceipt(state25, addr1, contract, mockEvent25, nil)
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(state25, 25))

	mockEvent30 := []*loomchain.EventData{
		{
			Topics:      []string{"topic1", "topic2", "topic3"},
			EncodedBody: []byte("height30"),
			Address:     contract.MarshalPB(),
		},
	}
	state30 := common.MockStateAt(state, 30)
	_, err = receiptHandler.CacheReceipt(state30, addr1, contract, mockEvent30, nil)
	require.NoError(t, err)
	receiptHandler.CommitCurrentReceipt()
	require.NoError(t, receiptHandler.CommitBlock(state30, 30))

	return state
}

func TestAddRemove(t *testing.T) {
	s := NewEthSubscriptions()

	myFilter := "{\"fromBlock\":\"0x0\",\"toBlock\":\"latest\",\"address\":\"\",\"topics\":[]}"
	id, err := s.AddLogPoll(myFilter, 1)
	require.NoError(t, err)
	_, ok := s.polls[id]
	require.True(t, ok, "map key does not exists")

	s.Remove(id)
	_, ok = s.polls[id]
	require.False(t, ok, "id key not deleted")
}
