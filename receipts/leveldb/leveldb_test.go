package leveldb

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain/receipts/common"
	evmaux "github.com/loomnetwork/loomchain/store/evm_aux"
	"github.com/stretchr/testify/require"
)

const (
	dbConfigKeys = 3
)

func TestReceiptsCyclicDB(t *testing.T) {
	evmAuxStore, err := common.NewMockEvmAuxStore()
	require.NoError(t, err)

	maxSize := uint64(10)
	handler := NewLevelDbReceipts(evmAuxStore, maxSize)
	// start db
	height := uint64(1)
	receipts1 := common.MakeDummyReceipts(t, 5, height)
	commit := 1 // number of commits
	// store 5 receipts
	require.NoError(t, handler.CommitBlock(receipts1, height))
	confirmDbConsistency(t, handler, 5, receipts1[0].TxHash, receipts1[4].TxHash, receipts1, commit)
	// db reaching max
	height = 2
	receipts2 := common.MakeDummyReceipts(t, 7, height)
	commit = 2
	// store another 7 receipts
	require.NoError(t, handler.CommitBlock(receipts2, height))
	confirmDbConsistency(t, handler, maxSize, receipts1[2].TxHash, receipts2[6].TxHash, append(receipts1[2:5], receipts2...), commit)

	// db at max
	height = 3
	receipts3 := common.MakeDummyReceipts(t, 5, height)
	commit = 3
	// store another 5 receipts
	require.NoError(t, handler.CommitBlock(receipts3, height))
	confirmDbConsistency(t, handler, maxSize, receipts2[2].TxHash, receipts3[4].TxHash, append(receipts2[2:7], receipts3...), commit)

	require.NoError(t, handler.Close())

	_, err = os.Stat(evmaux.EvmAuxDBName)
	require.NoError(t, err)
	handler.ClearData()
	_, err = os.Stat(evmaux.EvmAuxDBName)
	require.Error(t, err)
}

func TestReceiptsCommitAllInOneBlock(t *testing.T) {
	evmAuxStore, err := common.NewMockEvmAuxStore()
	require.NoError(t, err)

	maxSize := uint64(10)
	handler := NewLevelDbReceipts(evmAuxStore, maxSize)

	height := uint64(1)
	receipts1 := common.MakeDummyReceipts(t, maxSize+1, height)
	commit := 1
	// store 11 receipts, which is more than max that can be stored
	require.NoError(t, handler.CommitBlock(receipts1, height))

	confirmDbConsistency(t, handler, maxSize, receipts1[1].TxHash, receipts1[10].TxHash, receipts1[1:], commit)

	require.NoError(t, handler.Close())

	_, err = os.Stat(evmaux.EvmAuxDBName)
	require.NoError(t, err)
	handler.ClearData()
	_, err = os.Stat(evmaux.EvmAuxDBName)
	require.Error(t, err)
}

func confirmDbConsistency(t *testing.T, handler *LevelDbReceipts,
	size uint64, head, tail []byte, receipts []*types.EvmTxReceipt, commit int) {
	var err error
	dbSize, dbHead, dbTail, err := getDBParams(handler.evmAuxStore)
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

	_, err = os.Stat(evmaux.EvmAuxDBName)
	require.False(t, os.IsNotExist(err))

	for i := 0; i < len(receipts); i++ {
		getDBReceipt, err := handler.GetReceipt(receipts[i].TxHash)
		require.NoError(t, err)
		require.EqualValues(t, receipts[i].TransactionIndex, getDBReceipt.TransactionIndex)
		require.EqualValues(t, receipts[i].BlockNumber, getDBReceipt.BlockNumber)
		require.EqualValues(t, 0, bytes.Compare(receipts[i].TxHash, getDBReceipt.TxHash))
	}
	metadataCount := uint64(commit * 2)

	dbActualSize, err := countDbEntries(handler.evmAuxStore)
	require.NoError(t, err)
	require.EqualValues(t, size+dbConfigKeys+metadataCount, dbActualSize)

	require.EqualValues(t, 0, bytes.Compare(head, receipts[0].TxHash))
	require.EqualValues(t, 0, bytes.Compare(tail, receipts[len(receipts)-1].TxHash))

	previous := types.EvmTxReceiptListItem{}
	for i := 0; i < len(receipts); i++ {
		if previous.Receipt != nil {
			require.EqualValues(t, 0, bytes.Compare(receipts[i].TxHash, previous.NextTxHash))
		}
		txReceiptItemProto, err := handler.evmAuxStore.DB().Get(receipts[i].TxHash, nil)
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(txReceiptItemProto, &previous))
		require.EqualValues(t, 0, bytes.Compare(receipts[i].TxHash, previous.Receipt.TxHash))
	}
}

func TestConfirmTransactionReceipts(t *testing.T) {
	evmAuxStore, err := common.NewMockEvmAuxStore()
	require.NoError(t, err)
	maxSize := uint64(10)
	handler := NewLevelDbReceipts(evmAuxStore, maxSize)
	height := uint64(1)
	receipts1 := common.MakeDummyReceipts(t, 5, height)
	// store 5 receipts
	require.NoError(t, handler.CommitBlock(receipts1, height))
	require.NoError(t, handler.Close())
	_, err = os.Stat(evmaux.EvmAuxDBName)
	require.NoError(t, err)
	handler.ClearData()
	_, err = os.Stat(evmaux.EvmAuxDBName)
	require.Error(t, err)
}

//nolint:deadcode
func dumpDbEntries(evmAuxStore *evmaux.EvmAuxStore) error {
	fmt.Println("\nDumping leveldb")
	db := evmAuxStore.DB()
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		fmt.Printf("key %s\t\tvalue %s", iter.Key(), iter.Value())
	}
	fmt.Println()
	return iter.Error()
}

func countDbEntries(evmAuxStore *evmaux.EvmAuxStore) (uint64, error) {
	count := uint64(0)
	db := evmAuxStore.DB()
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		count++
	}
	return count, iter.Error()
}
