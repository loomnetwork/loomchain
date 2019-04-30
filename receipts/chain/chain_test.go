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

func TestConfirmTransactionReceipts(t *testing.T) {
	handler := StateDBReceipts{}
	handler.ClearData()
	// start db
	height := uint64(1)
	state := common.MockState(height)
	receipts1 := common.MakeDummyReceipts(t, 5, height)
	handler.CommitBlock(state, receipts1, height)
	txHashes, err := common.GetTxHashList(state, height)
	require.NoError(t, err)

	a := []byte("0xf0675dc27bC62b584Ab2E8E1D483a55CFac9E960")
	b := []byte("0xe288d6eec7150D6a22FDE33F0AA2d81E06591C4d")
	c := append(txHashes, a, b)

	for i := 0; i < len(c); i++ {
		//for i > len(c)-3 These are invalid tx hashes, so error must be returned by GetReceipt in this case
		if i > len(c)-3 {
			_, err := handler.GetReceipt(state, c[i])
			require.Error(t, err)
		} else {
			//These are valid hashes so valid txReceipt must be returned
			txReceipt, err := handler.GetReceipt(state, c[i])
			require.NoError(t, err)
			require.EqualValues(t, 0, bytes.Compare(c[i], txReceipt.TxHash))
		}

	}
}
