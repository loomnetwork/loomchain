package evmaux

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/plugin/types"
	"github.com/loomnetwork/go-loom/util"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tendermint/libs/db"
)

const (
	dbConfigKeys = 3
)

func TestLoadDupEvmTxHashes(t *testing.T) {

	evmAuxDB := dbm.NewMemDB()
	// load to set dup tx hashes
	evmAuxStore := NewEvmAuxStore(evmAuxDB, 10000)
	// add dup EVM txhash keys prefixed with dtx
	for i := 0; i < 100; i++ {
		evmAuxStore.db.Set(dupTxHashKey([]byte(fmt.Sprintf("hash:%d", i))), []byte{1})
	}
	// add 100 keys prefixed with hash
	for i := 0; i < 100; i++ {
		evmAuxStore.db.Set([]byte(fmt.Sprintf("hash:%d", i)), []byte{1})
	}
	// add another 100 keys prefixed with ahash
	for i := 0; i < 100; i++ {
		evmAuxStore.db.Set([]byte(fmt.Sprintf("ahash:%d", i)), []byte{1})
	}

	dupEVMTxHashes := make(map[string]bool)
	iter := evmAuxDB.Iterator(
		dupTxHashPrefix, util.PrefixRangeEnd(dupTxHashPrefix),
	)
	defer iter.Close()
	for iter.Valid() {
		dupTxHash, err := util.UnprefixKey(iter.Key(), dupTxHashPrefix)
		require.NoError(t, err)
		dupEVMTxHashes[string(dupTxHash)] = true
		iter.Next()
	}
	evmAuxStore.SetDupEVMTxHashes(dupEVMTxHashes)
	dupEvmTxHashes := evmAuxStore.GetDupEVMTxHashes()
	require.Equal(t, 100, len(dupEvmTxHashes))
}

func TestReceiptsCyclicDB(t *testing.T) {
	evmAuxStore := NewEvmAuxStore(dbm.NewMemDB(), 10)
	// start db
	height := uint64(1)
	receipts1 := makeDummyReceipts(t, 5, height)
	commit := 1 // number of commits
	// store 5 receipts
	require.NoError(t, evmAuxStore.CommitReceipts(receipts1, height))
	confirmDbConsistency(t, evmAuxStore, 5, receipts1[0].TxHash, receipts1[4].TxHash, receipts1, commit)
	confirmStateConsistency(t, evmAuxStore, receipts1, height)
	// db reaching max
	height = 2
	receipts2 := makeDummyReceipts(t, 7, height)
	commit = 2
	// store another 7 receipts
	require.NoError(t, evmAuxStore.CommitReceipts(receipts2, height))
	confirmDbConsistency(t, evmAuxStore, 10, receipts1[2].TxHash, receipts2[6].TxHash, append(receipts1[2:5], receipts2...), commit)
	confirmStateConsistency(t, evmAuxStore, receipts2, height)

	// db at max
	height = 3
	receipts3 := makeDummyReceipts(t, 5, height)
	commit = 3
	// store another 5 receipts
	require.NoError(t, evmAuxStore.CommitReceipts(receipts3, height))
	confirmDbConsistency(t, evmAuxStore, 10, receipts2[2].TxHash, receipts3[4].TxHash, append(receipts2[2:7], receipts3...), commit)
	confirmStateConsistency(t, evmAuxStore, receipts3, height)
}

func TestReceiptsCommitAllInOneBlock(t *testing.T) {
	maxSize := uint64(10)
	evmAuxStore := NewEvmAuxStore(dbm.NewMemDB(), maxSize)

	height := uint64(1)
	receipts1 := makeDummyReceipts(t, maxSize+1, height)
	commit := 1
	// store 11 receipts, which is more than max that can be stored
	require.NoError(t, evmAuxStore.CommitReceipts(receipts1, height))

	confirmDbConsistency(t, evmAuxStore, maxSize, receipts1[1].TxHash, receipts1[10].TxHash, receipts1[1:], commit)
	confirmStateConsistency(t, evmAuxStore, receipts1, height)

}

func confirmDbConsistency(t *testing.T, evmAuxStore *EvmAuxStore,
	size uint64, head, tail []byte, receipts []*types.EvmTxReceipt, commit int) {
	var err error
	dbSize, dbHead, dbTail, err := getDBParams(evmAuxStore.db)
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

	for i := 0; i < len(receipts); i++ {
		getDBReceipt, err := evmAuxStore.GetReceipt(receipts[i].TxHash)
		require.NoError(t, err)
		require.EqualValues(t, receipts[i].TransactionIndex, getDBReceipt.TransactionIndex)
		require.EqualValues(t, receipts[i].BlockNumber, getDBReceipt.BlockNumber)
		require.EqualValues(t, 0, bytes.Compare(receipts[i].TxHash, getDBReceipt.TxHash))
	}
	metadataCount := uint64(commit * 2)

	dbActualSize := countDbEntries(evmAuxStore)
	require.EqualValues(t, size+dbConfigKeys+metadataCount, dbActualSize)

	require.EqualValues(t, 0, bytes.Compare(head, receipts[0].TxHash))
	require.EqualValues(t, 0, bytes.Compare(tail, receipts[len(receipts)-1].TxHash))

	previous := types.EvmTxReceiptListItem{}
	for i := 0; i < len(receipts); i++ {
		if previous.Receipt != nil {
			require.EqualValues(t, 0, bytes.Compare(receipts[i].TxHash, previous.NextTxHash))
		}
		previous, err := evmAuxStore.GetReceipt(receipts[i].TxHash)
		require.NoError(t, err)
		require.EqualValues(t, 0, bytes.Compare(receipts[i].TxHash, previous.TxHash))
	}
}

func confirmStateConsistency(t *testing.T, evmAuxStore *EvmAuxStore, receipts []*types.EvmTxReceipt, height uint64) {
	txHashes, err := evmAuxStore.GetTxHashList(height)
	require.NoError(t, err)
	for i := 0; i < len(receipts); i++ {
		require.EqualValues(t, 0, bytes.Compare(txHashes[i], receipts[i].TxHash))
	}
}

func TestConfirmTransactionReceipts(t *testing.T) {
	maxSize := uint64(10)
	evmAuxStore := NewEvmAuxStore(dbm.NewMemDB(), maxSize)
	height := uint64(1)
	receipts1 := makeDummyReceipts(t, 5, height)
	// store 5 receipts
	require.NoError(t, evmAuxStore.CommitReceipts(receipts1, height))
	txHashes, err := evmAuxStore.GetTxHashList(height)
	require.NoError(t, err)
	a := []byte("0xf0675dc27bC62b584Ab2E8E1D483a55CFac9E960")
	b := []byte("0xe288d6eec7150D6a22FDE33F0AA2d81E06591C4d")
	c := append(txHashes, a, b)

	for i := 0; i < len(c); i++ {
		//for i > len(c)-3 These are invalid tx hashes, so error must be returned by GetReceipt in this case
		if i > len(c)-3 {
			_, err1 := evmAuxStore.GetReceipt(c[i])
			require.Error(t, err1)
		} else {
			//These are valid hashes so valid txReceipt must be returned
			txReceipt, err1 := evmAuxStore.GetReceipt(c[i])
			require.NoError(t, err1)
			require.EqualValues(t, 0, bytes.Compare(c[i], txReceipt.TxHash))
		}
	}
}

func countDbEntries(evmAuxStore *EvmAuxStore) uint64 {
	count := uint64(0)
	db := evmAuxStore.DB()
	iter := db.Iterator(nil, nil)
	defer iter.Close()
	for iter.Valid() {
		count++
		iter.Next()
	}
	return count
}

func makeDummyReceipts(t *testing.T, num, block uint64) []*types.EvmTxReceipt {
	var dummies []*types.EvmTxReceipt
	for i := uint64(0); i < num; i++ {
		dummy := types.EvmTxReceipt{
			TransactionIndex: int32(i),
			BlockNumber:      int64(block),
			Status:           statusTxSuccess,
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
