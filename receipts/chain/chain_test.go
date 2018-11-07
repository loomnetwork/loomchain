package chain

import (
	"bytes"
	"testing"

	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/stretchr/testify/require"
)

func TestReceiptsStateDB(t *testing.T) {
	handler := StateDBReceipts{}
	handler.ClearData()

	// start db
	height := uint64(1)
	state := common.MockState(height)
	receipts1 := common.MakeDummyReceipts(t, 5, height)
	handler.CommitBlock(state, receipts1, height)
	confirmStateConsistency(t, state, receipts1, height)

	// db reaching max
	height = 2
	state2 := common.MockStateAt(state, height)
	receipts2 := common.MakeDummyReceipts(t, 7, height)
	handler.CommitBlock(state2, receipts2, height)
	confirmStateConsistency(t, state2, receipts2, height)

	// db at max
	height = 3
	state3 := common.MockStateAt(state, height)
	receipts3 := common.MakeDummyReceipts(t, 5, height)
	handler.CommitBlock(state3, receipts3, height)
	confirmStateConsistency(t, state3, receipts3, height)
}

func TestUpdateReceipt(t *testing.T) {
	handler := StateDBReceipts{}
	handler.ClearData()

	height := uint64(1)
	state := common.MockState(height)
	receipt := common.MakeDummyReceipt(t, 0, 0,[]*types.EventData{})
	require.NoError(t, handler.CommitBlock(state, []*types.EvmTxReceipt{receipt}, height))

	receipt.BlockHash = []byte("myBlockHash")
	receipt.TransactionIndex = 12
	handler.UpdateReceipt(state, *receipt)
	updatedReceipt, err := handler.GetReceipt(state, receipt.TxHash)
	require.NoError(t, err)
	require.EqualValues(t, 0, bytes.Compare(receipt.BlockHash, updatedReceipt.BlockHash))
	require.EqualValues(t, 0, bytes.Compare(receipt.TxHash, updatedReceipt.TxHash))
	require.EqualValues(t, receipt.TransactionIndex, updatedReceipt.TransactionIndex)
}

func confirmStateConsistency(t *testing.T, state loomchain.State, receipts []*types.EvmTxReceipt, height uint64) {
	txHashes, err := common.GetTxHashList(state, height)
	require.NoError(t, err)
	handler := StateDBReceipts{}
	for i := 0; i < len(receipts); i++ {
		require.EqualValues(t, 0, bytes.Compare(txHashes[i], receipts[i].TxHash))

		getDBReceipt, err := handler.GetReceipt(state, txHashes[i])
		require.NoError(t, err)
		require.EqualValues(t, receipts[i].TransactionIndex, getDBReceipt.TransactionIndex)
		require.EqualValues(t, receipts[i].BlockNumber, getDBReceipt.BlockNumber)
		require.EqualValues(t, 0, bytes.Compare(receipts[i].TxHash, getDBReceipt.TxHash))
	}
}
