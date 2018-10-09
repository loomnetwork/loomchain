package leveldb

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
	
	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	"github.com/syndtr/goleveldb/leveldb"
	abci "github.com/tendermint/tendermint/abci/types"
)

const (
	dbConfigKeys = 3
)

func TestReceiptsCyclicDB(t *testing.T) {
	maxSize := uint64(10)
	handler := LevelDbReceipts{maxSize}
	handler.ClearData()
	_, err := os.Stat(Db_Filename)
	require.True(t,os.IsNotExist(err))
	
	// start db
	height := uint64(1)
	state := mockState(height)
	receipts1 := makeDummyReceipts(t, 5, height)
	handler.CommitBlock(state, receipts1, height)
	confirmDbConsistency(t, 5, receipts1[0].TxHash, receipts1[4].TxHash, receipts1)
	confirmStateConsistency(t, state, receipts1, height)
	
	// db reaching max
	height = 2
	state2 := mockStateAt(state, height)
	receipts2 := makeDummyReceipts(t, 7, height)
	handler.CommitBlock(state2, receipts2, height)
	confirmDbConsistency(t, maxSize, receipts1[2].TxHash, receipts2[6].TxHash, append(receipts1[2:5], receipts2...))
	confirmStateConsistency(t, state2, receipts2, height)
	
	// db at max
	height = 3
	state3 := mockStateAt(state, height)
	receipts3 := makeDummyReceipts(t, 5, height)
	handler.CommitBlock(state3, receipts3, height)
	confirmDbConsistency(t, maxSize, receipts2[2].TxHash, receipts3[4].TxHash, append(receipts2[2:7], receipts3...))
	confirmStateConsistency(t, state3, receipts3, height)
	
	_, err = os.Stat(Db_Filename)
	require.NoError(t, err)
	handler.ClearData()
	_, err = os.Stat(Db_Filename)
	require.Error(t, err)
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

func confirmDbConsistency(t *testing.T, size uint64, head, tail []byte, receipts []*types.EvmTxReceipt) {
	var err error
	require.EqualValues(t, size, uint64(len(receipts)))
	if ( size == 0 ) {
		require.EqualValues(t, 0 ,len(head))
		require.EqualValues(t, 0 ,len(tail))
		return
	}
	_, err = os.Stat(Db_Filename)
	require.False(t,os.IsNotExist(err))
	
	handler := LevelDbReceipts {uint64(len(receipts)+1)	}
	for i := 0 ; i < len(receipts) ; i++ {
		getDBReceipt, err := handler.GetReceipt(receipts[i].TxHash)
		require.NoError(t, err)
		require.EqualValues(t, receipts[i].TransactionIndex, getDBReceipt.TransactionIndex)
		require.EqualValues(t, receipts[i].BlockNumber, getDBReceipt.BlockNumber)
		require.EqualValues(t, string(receipts[i].TxHash), string(getDBReceipt.TxHash))
	}
	
	db, err := leveldb.OpenFile(Db_Filename, nil)
	defer db.Close()
	require.NoError(t, err)
	
	dbSize, dbHead, dbTail, err := getDBParams(db)
	require.NoError(t, err)
	
	require.EqualValues(t, size, dbSize)

	dbActualSize, err := countDbEntries(db)
	require.EqualValues(t, size + dbConfigKeys, dbActualSize)
	
	require.EqualValues(t, string(head), string(receipts[0].TxHash))
	require.EqualValues(t, string(tail), string((receipts[len(receipts)-1].TxHash)))

	require.EqualValues(t, string(dbHead), head)
	require.EqualValues(t, string(dbTail), tail)
	
	previous := types.EvmTxReceiptListItem{}
	for i := 0 ; i < len(receipts) ; i++ {
		if previous.Receipt != nil {
			require.EqualValues(t, string(receipts[i].TxHash), string(previous.NextTxHash))
		}
		txReceiptItemProto, err := db.Get(receipts[i].TxHash, nil)
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

func dumpDbEntries(db *leveldb.DB) error {
	fmt.Println("\nDumping leveldb\n\n")
	iter := db.NewIterator(nil, nil)
	defer iter.Release()
	for iter.Next() {
		fmt.Printf("key %s\t\tvalue %s\n", string(iter.Key()), string(iter.Value()))
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