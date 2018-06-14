// +build evm

package subs

import (
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ptypes "github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/eth/query"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPoll(t *testing.T) {
	sub := NewEthSubscriptions()
	allFilter := "{\"fromBlock\":\"0x0\",\"toBlock\":\"pending\",\"address\":\"\",\"topics\":[]}"
	state := makeMockState(t)
	id, err := sub.Add(allFilter)
	require.NoError(t, err)

	state5 := query.MockStateAt(state, int64(5))
	result, err := sub.Poll(state5, id)
	require.NoError(t, err)
	var logs ptypes.EthFilterLogList
	require.NoError(t, proto.Unmarshal(result, &logs), "unmarshalling EthFilterLogList")
	require.Equal(t, 1, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.Equal(t, "height4", string(logs.EthBlockLogs[0].Data))

	state40 := query.MockStateAt(state, int64(40))
	result, err = sub.Poll(state40, id)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &logs), "unmarshalling EthFilterLogList")
	require.Equal(t, 3, len(logs.EthBlockLogs), "wrong number of logs returned")
	require.Equal(t, "height20", string(logs.EthBlockLogs[0].Data))
	require.Equal(t, "height25", string(logs.EthBlockLogs[1].Data))
	require.Equal(t, "height30", string(logs.EthBlockLogs[2].Data))

	state50 := query.MockStateAt(state, int64(50))
	result, err = sub.Poll(state50, id)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(result, &logs), "unmarshalling EthFilterLogList")
	require.Equal(t, 0, len(logs.EthBlockLogs), "wrong number of logs returned")

	state60 := query.MockStateAt(state, int64(60))
	sub.Remove(id)
	result, err = sub.Poll(state60, id)
	require.Error(t, err, "subscription not removed")
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
	id, err := s.Add(myFilter)
	require.NoError(t, err)
	_, ok := s.subs[id]
	require.True(t, ok, "map key does not exists")
	require.Equal(t, "0x0", s.subs[id].filter.FromBlock)
	require.Equal(t, "latest", s.subs[id].filter.ToBlock)

	s.Remove(id)
	_, ok = s.subs[id]
	require.False(t, ok, "id key not deleted")
}
