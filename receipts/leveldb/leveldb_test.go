package leveldb

import (
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
	require.True(t,os.IsNotExist(err))
	
	maxSize := uint64(10)
	handler, err := NewLevelDbReceipts(maxSize)
	require.NoError(t, err)
	
	// start db
	height := uint64(1)
	state := common.MockState(height)
	receipts1 := common.MakeDummyReceipts(t, 5, height)
	require.NoError(t, handler.CommitBlock(state, receipts1, height))
	confirmDbConsistency(t, handler, 5, receipts1[0].TxHash, receipts1[4].TxHash, receipts1)
	confirmStateConsistency(t, state, receipts1, height)
	
	// db reaching max
	height = 2
	state2 := common.MockStateAt(state, height)
	receipts2 := common.MakeDummyReceipts(t, 7, height)
	require.NoError(t, handler.CommitBlock(state2, receipts2, height))
	confirmDbConsistency(t, handler, maxSize, receipts1[2].TxHash, receipts2[6].TxHash, append(receipts1[2:5], receipts2...))
	confirmStateConsistency(t, state2, receipts2, height)
	
	// db at max
	height = 3
	state3 := common.MockStateAt(state, height)
	receipts3 := common.MakeDummyReceipts(t, 5, height)
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



func confirmDbConsistency(t *testing.T, handler *LevelDbReceipts, size uint64, head, tail []byte, receipts []*types.EvmTxReceipt) {
	var err error
	dbSize, dbHead, dbTail, err := getDBParams(handler.db)
	require.NoError(t, err)
	
	require.EqualValues(t, size, uint64(len(receipts)))
	require.EqualValues(t, size, dbSize)
	require.EqualValues(t, string(dbHead), head)
	require.EqualValues(t, string(dbTail), tail)
	if ( size == 0 ) {
		require.EqualValues(t, 0 ,len(head))
		require.EqualValues(t, 0 ,len(tail))
		return
	}
	
	_, err = os.Stat(Db_Filename)
	require.False(t,os.IsNotExist(err))
	
	for i := 0 ; i < len(receipts) ; i++ {
		getDBReceipt, err := handler.GetReceipt(receipts[i].TxHash)
		require.NoError(t, err)
		require.EqualValues(t, receipts[i].TransactionIndex, getDBReceipt.TransactionIndex)
		require.EqualValues(t, receipts[i].BlockNumber, getDBReceipt.BlockNumber)
		require.EqualValues(t, string(receipts[i].TxHash), string(getDBReceipt.TxHash))
	}

	dbActualSize, err := countDbEntries(handler.db)
	require.EqualValues(t, size + dbConfigKeys, dbActualSize)
	
	require.EqualValues(t, string(head), string(receipts[0].TxHash))
	require.EqualValues(t, string(tail), string((receipts[len(receipts)-1].TxHash)))
	
	previous := types.EvmTxReceiptListItem{}
	for i := 0 ; i < len(receipts) ; i++ {
		if previous.Receipt != nil {
			require.EqualValues(t, string(receipts[i].TxHash), string(previous.NextTxHash))
		}
		txReceiptItemProto, err := handler.db.Get(receipts[i].TxHash, nil)
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(txReceiptItemProto, &previous))
		require.EqualValues(t, string(receipts[i].TxHash), string(previous.Receipt.TxHash))
	}
}

func confirmStateConsistency(t *testing.T,state loomchain.State, receipts []*types.EvmTxReceipt, height uint64){
	txHashes, err := common.GetTxHashList(state, height)
	require.NoError(t, err)
	for i := 0 ; i < len(receipts) ; i++ {
		require.EqualValues(t, string(txHashes[i]), string(receipts[i].TxHash))
	}
}

func dumpDbEntries(db *leveldb.DB) error {
	fmt.Println("\nDumping leveldb")
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		fmt.Printf("key %s\t\tvalue %s", string(iter.Key()), string(iter.Value()))
	}
	fmt.Println("\n")
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