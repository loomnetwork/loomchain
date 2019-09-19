package common

import (
	"context"
	"crypto/sha256"
	"os"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/stretchr/testify/require"
	goleveldb "github.com/syndtr/goleveldb/leveldb"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
)

func MakeDummyReceipts(t *testing.T, num, block uint64) []*types.EvmTxReceipt {
	var dummies []*types.EvmTxReceipt
	for i := uint64(0); i < num; i++ {
		dummy := types.EvmTxReceipt{
			TransactionIndex: int32(i),
			BlockNumber:      int64(block),
			Status:           StatusTxSuccess,
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
		TransactionIndex: int32(txNum),
		BlockNumber:      int64(block),
		Status:           StatusTxSuccess,
	}
	protoDummy, err := proto.Marshal(&dummy)
	require.NoError(t, err)
	h := sha256.New()
	h.Write(protoDummy)
	dummy.TxHash = h.Sum(nil)
	dummy.Logs = events

	return &dummy
}

func MockState(height uint64) state.State {
	header := abci.Header{}
	header.Height = int64(height)
	return state.NewStoreState(context.Background(), store.NewMemStore(), header, nil, nil)
}

func MockStateTx(s state.State, height, TxNum uint64) state.State {
	header := abci.Header{}
	header.Height = int64(height)
	header.NumTxs = int64(TxNum)
	return state.NewStoreState(context.Background(), s, header, nil, nil)
}

func MockStateAt(s state.State, newHeight uint64) state.State {
	header := abci.Header{}
	header.Height = int64(newHeight)
	return state.NewStoreState(context.Background(), s, header, nil, nil)
}

func NewMockEvmAuxStore() (*evmaux.EvmAuxStore, error) {
	os.RemoveAll(evmaux.EvmAuxDBName)
	evmAuxDB, err := goleveldb.OpenFile(evmaux.EvmAuxDBName, nil)
	if err != nil {
		return nil, err
	}
	evmAuxStore := evmaux.NewEvmAuxStore(evmAuxDB)
	return evmAuxStore, nil
}
