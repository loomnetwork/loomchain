// +build evm

package polls

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/query"
	"github.com/loomnetwork/loomchain/receipts/factory"
	"github.com/stretchr/testify/require"
)

func TestLogPoll(t *testing.T) {
	rhFactory, err := factory.NewReceiptHandlerFactory(factory.ReceiptHandlerChain, &loomchain.DefaultEventHandler{})
	require.NoError(t, err)

	sub := NewEthSubscriptions()
	allFilter := "{\"fromBlock\":\"0x0\",\"toBlock\":\"pending\",\"address\":\"\",\"topics\":[]}"
	state := makeMockState(t)
	id, err := sub.AddLogPoll(allFilter, 1)
	require.NoError(t, err)

	state5 := query.MockStateAt(state, int64(5))
	result, err := sub.Poll(state5, id, rhFactory)
	require.NoError(t, err)

	var envolope ptypes.EthFilterEnvelope
	var logs *ptypes.EthFilterLogList
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 1, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.Equal(t, "height4", string(logs.EthBlockLogs[0].Data))

	state40 := query.MockStateAt(state, int64(40))
	result, err = sub.Poll(state40, id, rhFactory)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 3, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.Equal(t, "height20", string(logs.EthBlockLogs[0].Data))
	require.Equal(t, "height25", string(logs.EthBlockLogs[1].Data))
	require.Equal(t, "height30", string(logs.EthBlockLogs[2].Data))

	state50 := query.MockStateAt(state, int64(50))
	result, err = sub.Poll(state50, id, rhFactory)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 0, len(logs.EthBlockLogs), "wrong number of logs returned")

	state60 := query.MockStateAt(state, int64(60))
	sub.Remove(id)
	result, err = sub.Poll(state60, id, rhFactory)
	require.Error(t, err, "subscription not removed")
}

func TestTxPoll(t *testing.T) {

	rhFactory, err := factory.NewReceiptHandlerFactory(factory.ReceiptHandlerChain, &loomchain.DefaultEventHandler{})
	sub := NewEthSubscriptions()
	state := makeMockState(t)
	id := sub.AddTxPoll(uint64(5))

	var envolope ptypes.EthFilterEnvelope
	var txHashes *ptypes.EthTxHashList
	state27 := query.MockStateAt(state, int64(27))
	result, err := sub.Poll(state27, id, rhFactory)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 2, len(txHashes.EthTxHash), "wrong number of logs returned")

	state50 := query.MockStateAt(state, int64(50))
	result, err = sub.Poll(state50, id, rhFactory)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 1, len(txHashes.EthTxHash), "wrong number of logs returned")

	state60 := query.MockStateAt(state, int64(60))
	sub.Remove(id)
	result, err = sub.Poll(state60, id, rhFactory)
	require.Error(t, err, "subscription not removed")
}

func TestTimeout(t *testing.T) {
	rhFactory, err := factory.NewReceiptHandlerFactory(factory.ReceiptHandlerChain, &loomchain.DefaultEventHandler{})
	BlockTimeout = 10
	sub := NewEthSubscriptions()
	state := makeMockState(t)

	var envolope ptypes.EthFilterEnvelope
	var txHashes *ptypes.EthTxHashList
	id := sub.AddTxPoll(uint64(1))

	state5 := query.MockStateAt(state, int64(5))
	_ = sub.AddTxPoll(uint64(5))

	result, err := sub.Poll(state5, id, rhFactory)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 1, len(txHashes.EthTxHash), "wrong number of logs returned")

	state12 := query.MockStateAt(state, int64(12))
	_ = sub.AddTxPoll(uint64(12))

	result, err = sub.Poll(state12, id, rhFactory)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 0, len(txHashes.EthTxHash), "wrong number of logs returned")

	state40 := query.MockStateAt(state, int64(40))
	_ = sub.AddTxPoll(uint64(40))

	result, err = sub.Poll(state40, id, rhFactory)
	require.Error(t, err, "poll did not timed out")
}

func makeMockState(t *testing.T) loomchain.State {
	contract, err := loom.LocalAddressFromHexString("0x1234567890123456789012345678901234567890")
	require.NoError(t, err)
	receipts := []query.MockReceipt{
		{
			Height:   uint64(4),
			Contract: contract,
			Events: []query.MockEvent{
				{
					Topics: []string{"topic1", "topic2", "topic3"},
					Data:   []byte("height4"),
				},
			},
		},
		{
			Height:   uint64(20),
			Contract: contract,
			Events: []query.MockEvent{
				{
					Topics: []string{"topic1"},
					Data:   []byte("height20"),
				},
			},
		},
		{
			Height:   uint64(25),
			Contract: contract,
			Events: []query.MockEvent{
				{
					Topics: []string{"topic1"},
					Data:   []byte("height25"),
				},
			},
		},
		{
			Height:   uint64(30),
			Contract: contract,
			Events: []query.MockEvent{
				{
					Topics: []string{"topic1", "topic2", "topic3"},
					Data:   []byte("height30"),
				},
			},
		},
	}
	state, err := query.MockPopulatedState(receipts)
	require.NoError(t, err)
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
