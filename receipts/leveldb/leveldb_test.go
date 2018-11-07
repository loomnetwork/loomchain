package leveldb

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	dbConfigKeys = 3
)

func TestReceiptsCyclicDB(t *testing.T) {
	os.RemoveAll(Db_Filename)
	_, err := os.Stat(Db_Filename)
	require.True(t, os.IsNotExist(err))

	maxSize := uint64(10)
	handler, err := NewLevelDbReceipts(maxSize)
	require.NoError(t, err)

	// start db
	height := uint64(1)
	state := common.MockState(height)
	receipts1 := common.MakeDummyReceipts(t, 5, height)
	// store 5 receipts
	require.NoError(t, handler.CommitBlock(state, receipts1, height))
	confirmDbConsistency(t, handler, 5, receipts1[0].TxHash, receipts1[4].TxHash, receipts1)
	confirmStateConsistency(t, state, receipts1, height)

	// db reaching max
	height = 2
	state2 := common.MockStateAt(state, height)
	receipts2 := common.MakeDummyReceipts(t, 7, height)
	// store another 7 receipts
	require.NoError(t, handler.CommitBlock(state2, receipts2, height))
	confirmDbConsistency(t, handler, maxSize, receipts1[2].TxHash, receipts2[6].TxHash, append(receipts1[2:5], receipts2...))
	confirmStateConsistency(t, state2, receipts2, height)

	// db at max
	height = 3
	state3 := common.MockStateAt(state, height)
	receipts3 := common.MakeDummyReceipts(t, 5, height)
	// store another 5 receipts
	require.NoError(t, handler.CommitBlock(state3, receipts3, height))
	confirmDbConsistency(t, handler, maxSize, receipts2[2].TxHash, receipts3[4].TxHash, append(receipts2[2:7], receipts3...))
	confirmStateConsistency(t, state3, receipts3, height)

	require.NoError(t, handler.Close())

	_, err = os.Stat(Db_Filename)
	require.NoError(t, err)
	handler.ClearData()
	_, err = os.Stat(Db_Filename)
	require.Error(t, err)
}

func TestReceiptsCommitAllInOneBlock(t *testing.T) {
	os.RemoveAll(Db_Filename)
	_, err := os.Stat(Db_Filename)
	require.True(t, os.IsNotExist(err))

	maxSize := uint64(10)
	handler, err := NewLevelDbReceipts(maxSize)
	require.NoError(t, err)

	height := uint64(1)
	state := common.MockState(height)
	receipts1 := common.MakeDummyReceipts(t, maxSize+1, height)
	// store 11 receipts, which is more than max that can be stored
	require.NoError(t, handler.CommitBlock(state, receipts1, height))

	confirmDbConsistency(t, handler, maxSize, receipts1[1].TxHash, receipts1[10].TxHash, receipts1[1:])
	confirmStateConsistency(t, state, receipts1, height)

	require.NoError(t, handler.Close())

	_, err = os.Stat(Db_Filename)
	require.NoError(t, err)
	handler.ClearData()
	_, err = os.Stat(Db_Filename)
	require.Error(t, err)
}

func TestUpdateReceipt(t *testing.T) {
	os.RemoveAll(Db_Filename)
	_, err := os.Stat(Db_Filename)
	require.True(t, os.IsNotExist(err))

	maxSize := uint64(10)
	handler, err := NewLevelDbReceipts(maxSize)
	require.NoError(t, err)

	height := uint64(1)
	state := common.MockState(height)
	receipt := common.MakeDummyReceipt(t, 0, 0,[]*types.EventData{})
	require.NoError(t, handler.CommitBlock(state, []*types.EvmTxReceipt{receipt}, height))

	receipt.BlockHash = []byte("myBlockHash")
	receipt.TransactionIndex = 12

	oldReceipt, err := handler.GetReceipt(receipt.TxHash)
	require.NoError(t, err)
	require.NotEqual(t, 0, bytes.Compare(receipt.BlockHash, oldReceipt.BlockHash))
	require.EqualValues(t, 0, bytes.Compare(receipt.TxHash, oldReceipt.TxHash))
	require.NotEqual(t, receipt.TransactionIndex, oldReceipt.TransactionIndex)

	handler.UpdateReceipt(*receipt)

	updatedReceipt, err := handler.GetReceipt(receipt.TxHash)
	require.NoError(t, err)
	require.EqualValues(t, 0, bytes.Compare(receipt.BlockHash, updatedReceipt.BlockHash))
	require.EqualValues(t, 0, bytes.Compare(receipt.TxHash, updatedReceipt.TxHash))
	require.EqualValues(t, receipt.TransactionIndex, updatedReceipt.TransactionIndex)

	_, err = os.Stat(Db_Filename)
	require.NoError(t, err)
	handler.ClearData()
	_, err = os.Stat(Db_Filename)
	require.Error(t, err)
}

func confirmDbConsistency(t *testing.T, handler *LevelDbReceipts, size uint64, head, tail []byte, receipts []*types.EvmTxReceipt) {
	var err error
	dbSize, dbHead, dbTail, err := getDBParams(handler.db)
	require.NoError(t, err)

	require.EqualValues(t, size, uint64(len(receipts)))
	require.EqualValues(t, size, dbSize)
	require.EqualValues(t, 0, bytes.Compare(dbHead, head))
	require.EqualValues(t, 0, bytes.Compare(dbTail, tail))
	if size == 0 {
		require.EqualValues(t, 0, len(head))
		require.EqualValues(t, 0, len(tail))
		return
	}

	_, err = os.Stat(Db_Filename)
	require.False(t, os.IsNotExist(err))

	for i := 0; i < len(receipts); i++ {
		getDBReceipt, err := handler.GetReceipt(receipts[i].TxHash)
		require.NoError(t, err)
		require.EqualValues(t, receipts[i].TransactionIndex, getDBReceipt.TransactionIndex)
		require.EqualValues(t, receipts[i].BlockNumber, getDBReceipt.BlockNumber)
		require.EqualValues(t, 0, bytes.Compare(receipts[i].TxHash, getDBReceipt.TxHash))
	}

	dbActualSize, err := countDbEntries(handler.db)
	require.EqualValues(t, size+dbConfigKeys, dbActualSize)

	require.EqualValues(t, 0, bytes.Compare(head, receipts[0].TxHash))
	require.EqualValues(t, 0, bytes.Compare(tail, receipts[len(receipts)-1].TxHash))

	previous := types.EvmTxReceiptListItem{}
	for i := 0; i < len(receipts); i++ {
		if previous.Receipt != nil {
			require.EqualValues(t, 0, bytes.Compare(receipts[i].TxHash, previous.NextTxHash))
		}
		txReceiptItemProto, err := handler.db.Get(receipts[i].TxHash, nil)
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(txReceiptItemProto, &previous))
		require.EqualValues(t, 0, bytes.Compare(receipts[i].TxHash, previous.Receipt.TxHash))
	}
}

func confirmStateConsistency(t *testing.T, state loomchain.State, receipts []*types.EvmTxReceipt, height uint64) {
	txHashes, err := common.GetTxHashList(state, height)
	require.NoError(t, err)
	for i := 0; i < len(receipts); i++ {
		require.EqualValues(t, 0, bytes.Compare(txHashes[i], receipts[i].TxHash))
	}
}

func dumpDbEntries(db *leveldb.DB) error {
	fmt.Println("\nDumping leveldb")
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		fmt.Printf("key %s\t\tvalue %s", iter.Key(), iter.Value())
	}
	fmt.Println()
	return iter.Error()
}

func countDbEntries(db *leveldb.DB) (uint64, error) {
	count := uint64(0)
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		count++
	}
	return count, iter.Error()
}
