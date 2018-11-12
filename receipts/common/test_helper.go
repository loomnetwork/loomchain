package common

import (
	"context"
	"crypto/sha256"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func MakeDummyReceipts(t *testing.T, num, block uint64) []*types.EvmTxReceipt {
	var dummies []*types.EvmTxReceipt
	for i := uint64(0); i < num; i++ {
		dummy := types.EvmTxReceipt{
			Nonce:          int64(i),
			BlockNumber:    int64(block),
			Status:         loomchain.StatusTxSuccess,
		}
		protoDummy, err := proto.Marshal(&dummy)
		require.NoError(t, err)
		h := sha256.New()
		h.Write(protoDummy)
		dummy.TxHash = h.Sum(nil)

		dummies = append(dummies, &dummy)
	}
	return dummies
}

func MakeDummyReceipt(t *testing.T, block, txNum uint64, events []*types.EventData) *types.EvmTxReceipt {
	dummy := types.EvmTxReceipt{
		Nonce: int64(txNum),
		BlockNumber:      int64(block),
	}
	protoDummy, err := proto.Marshal(&dummy)
	require.NoError(t, err)
	h := sha256.New()
	h.Write(protoDummy)
	dummy.TxHash = h.Sum(nil)
	dummy.Logs = events
	dummy.Status = loomchain.StatusTxSuccess

	return &dummy
}

func MockState(height uint64) loomchain.State {
	header := abci.Header{}
	header.Height = int64(height)
	return loomchain.NewStoreState(context.Background(), store.NewMemStore(), header)
}

func MockStateTx(state loomchain.State, height, TxNum uint64) loomchain.State {
	header := abci.Header{}
	header.Height = int64(height)
	header.NumTxs = int32(TxNum)
	return loomchain.NewStoreState(context.Background(), state, header)
}

func MockStateAt(state loomchain.State, newHeight uint64) loomchain.State {
	header := abci.Header{}
	header.Height = int64(newHeight)
	return loomchain.NewStoreState(context.Background(), state, header)
}
