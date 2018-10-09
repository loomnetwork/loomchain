package chain

import (
	"context"
	"crypto/sha256"
	"testing"
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestReceiptsStateDB(t *testing.T) {
	handler := StateDBReceipts{}
	handler.ClearData()

	
	// start db
	height := uint64(1)
	state := mockState(height)
	receipts1 := makeDummyReceipts(t, 5, height)
	handler.CommitBlock(state, receipts1, height)
	confirmStateConsistency(t, state, receipts1, height)
	
	// db reaching max
	height = 2
	state2 := mockStateAt(state, height)
	receipts2 := makeDummyReceipts(t, 7, height)
	handler.CommitBlock(state2, receipts2, height)
	confirmStateConsistency(t, state2, receipts2, height)
	
	// db at max
	height = 3
	state3 := mockStateAt(state, height)
	receipts3 := makeDummyReceipts(t, 5, height)
	handler.CommitBlock(state3, receipts3, height)
	confirmStateConsistency(t, state3, receipts3, height)
	
}

func confirmStateConsistency(t *testing.T,state loomchain.State, receipts []*types.EvmTxReceipt, height uint64){
	txHashes, err := common.GetTxHashList(state, height)
	require.NoError(t, err)
	handler := StateDBReceipts{}
	for i := 0 ; i < len(receipts) ; i++ {
		require.EqualValues(t, string(txHashes[i]), string(receipts[i].TxHash))
		
		getDBReceipt, err := handler.GetReceipt(state, txHashes[i])
		require.NoError(t, err)
		require.EqualValues(t, receipts[i].TransactionIndex, getDBReceipt.TransactionIndex)
		require.EqualValues(t, receipts[i].BlockNumber, getDBReceipt.BlockNumber)
		require.EqualValues(t, string(receipts[i].TxHash), string(getDBReceipt.TxHash))
	}
}

func mockState(height uint64) loomchain.State {
	header := abci.Header{}
	header.Height = int64(height)
	return loomchain.NewStoreState(context.Background(), store.NewMemStore(), header)
}

func mockStateAt(state loomchain.State, newHeight uint64) loomchain.State {
	header := abci.Header{}
	header.Height = int64(newHeight)
	return loomchain.NewStoreState(context.Background(), state, header)
}

func makeDummyReceipts(t *testing.T, num, block uint64) []*types.EvmTxReceipt {
	var dummies []*types.EvmTxReceipt
	for i := uint64(0) ; i < num ; i++ {
		dummy := types.EvmTxReceipt{
			TransactionIndex: int32(i),
			BlockNumber: int64(block),
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