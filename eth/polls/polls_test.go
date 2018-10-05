// +build evm

package polls

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	ctypes `github.com/loomnetwork/go-loom/builtin/types/config`
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/query"
	`github.com/loomnetwork/loomchain/receipts/factory`
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLogPoll(t *testing.T) {
	rhFactory, err := factory.NewReadReceiptHandlerFactory(ctypes.ReceiptStorage_CHAIN)
	require.NoError(t, err)
	
	sub := NewEthSubscriptions()
	allFilter := "{\"fromBlock\":\"0x0\",\"toBlock\":\"pending\",\"address\":\"\",\"topics\":[]}"
	state := makeMockState(t)
	id, err := sub.AddLogPoll(allFilter, 1)
	require.NoError(t, err)

	state5 := query.MockStateAt(state, int64(5))
	rh5, err := rhFactory(state5)
	require.NoError(t, err)
	result, err := sub.Poll(state5, id, rh5)
	require.NoError(t, err)

	var envolope ptypes.EthFilterEnvelope
	var logs *ptypes.EthFilterLogList
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 1, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.Equal(t, "height4", string(logs.EthBlockLogs[0].Data))

	state40 := query.MockStateAt(state, int64(40))
	rh40, err := rhFactory(state40)
	require.NoError(t, err)
	result, err = sub.Poll(state40, id, rh40)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 3, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.Equal(t, "height20", string(logs.EthBlockLogs[0].Data))
	require.Equal(t, "height25", string(logs.EthBlockLogs[1].Data))
	require.Equal(t, "height30", string(logs.EthBlockLogs[2].Data))

	state50 := query.MockStateAt(state, int64(50))
	rh50, err := rhFactory(state50)
	require.NoError(t, err)
	result, err = sub.Poll(state50, id, rh50)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	logs = envolope.GetEthFilterLogList()
	require.NotEqual(t, nil, logs)
	require.Equal(t, 0, len(logs.EthBlockLogs), "wrong number of logs returned")
	

	state60 := query.MockStateAt(state, int64(60))
	sub.Remove(id)
	rh60, err := rhFactory(state60)
	require.NoError(t, err)
	result, err = sub.Poll(state60, id, rh60)
	require.Error(t, err, "subscription not removed")
}

func TestTxPoll(t *testing.T) {
	rhFactory, err := factory.NewReadReceiptHandlerFactory(ctypes.ReceiptStorage_CHAIN)
	sub := NewEthSubscriptions()
	state := makeMockState(t)
	id := sub.AddTxPoll(uint64(5))

	var envolope ptypes.EthFilterEnvelope
	var txHashes *ptypes.EthTxHashList
	state27 := query.MockStateAt(state, int64(27))
	rh27, err := rhFactory(state27)
	require.NoError(t, err)
	result, err := sub.Poll(state27, id, rh27)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 2, len(txHashes.EthTxHash), "wrong number of logs returned")

	state50 := query.MockStateAt(state, int64(50))
	rh50, err := rhFactory(state50)
	require.NoError(t, err)
	result, err = sub.Poll(state50, id, rh50)
	require.NoError(t, err)

	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 1, len(txHashes.EthTxHash), "wrong number of logs returned")

	state60 := query.MockStateAt(state, int64(60))
	sub.Remove(id)
	rh60, err := rhFactory(state60)
	require.NoError(t, err)
	result, err = sub.Poll(state60, id, rh60)
	require.Error(t, err, "subscription not removed")
}

func TestTimeout(t *testing.T) {
	rhFactory, err := factory.NewReadReceiptHandlerFactory(ctypes.ReceiptStorage_CHAIN)
	BlockTimeout = 10
	sub := NewEthSubscriptions()
	state := makeMockState(t)

	var envolope ptypes.EthFilterEnvelope
	var txHashes *ptypes.EthTxHashList
	id := sub.AddTxPoll(uint64(1))

	state5 := query.MockStateAt(state, int64(5))
	_ = sub.AddTxPoll(uint64(5))
	
	rh5, err := rhFactory(state5)
	require.NoError(t, err)
	result, err := sub.Poll(state5, id, rh5)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 1, len(txHashes.EthTxHash), "wrong number of logs returned")

	state12 := query.MockStateAt(state, int64(12))
	_ = sub.AddTxPoll(uint64(12))
	
	rh12, err := rhFactory(state12)
	require.NoError(t, err)
	result, err = sub.Poll(state12, id, rh12)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &envolope), "unmarshalling EthFilterEnvelope")
	txHashes = envolope.GetEthTxHashList()
	require.NotEqual(t, nil, txHashes)
	require.Equal(t, 0, len(txHashes.EthTxHash), "wrong number of logs returned")
	
	state40 := query.MockStateAt(state, int64(40))
	_ = sub.AddTxPoll(uint64(40))
	
	rh40, err := rhFactory(state40)
	require.NoError(t, err)
	result, err = sub.Poll(state40, id, rh40)
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
