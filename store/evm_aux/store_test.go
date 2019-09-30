package evmaux

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadDupEvmTxHashes(t *testing.T) {

	// load to set dup tx hashes
	evmAuxStore, err := LoadStore()
	defer evmAuxStore.ClearData()
	require.NoError(t, err)
	for i := 0; i < 100; i++ {
		evmAuxStore.db.Put(dupTxHashKey([]byte(fmt.Sprintf("hash:%d", i))), []byte{1}, nil)
	}
	for i := 0; i < 100; i++ {
		evmAuxStore.db.Put([]byte(fmt.Sprintf("hash:%d", i)), []byte{1}, nil)
	}

	evmAuxStore2, err := LoadStore()
	defer evmAuxStore2.ClearData()
	dupEvmTxHashes := evmAuxStore2.GetDupEVMTxHashes()
	require.Equal(t, 100, len(dupEvmTxHashes))
}

func TestTxHashOperation(t *testing.T) {
	txHashList1 := [][]byte{
		[]byte("hash1"),
		[]byte("hash2"),
	}
	evmAuxStore, err := LoadStore()
	require.NoError(t, err)
	txHashList, err := evmAuxStore.GetTxHashList(40)
	require.NoError(t, err)
	require.Equal(t, 0, len(txHashList))
	db := evmAuxStore.DB()
	tran, err := db.OpenTransaction()
	require.NoError(t, err)
	evmAuxStore.SetTxHashList(tran, txHashList1, 30)
	tran.Commit()
	txHashList, err = evmAuxStore.GetTxHashList(30)
	require.NoError(t, err)
	require.Equal(t, 2, len(txHashList))
	require.Equal(t, true, bytes.Equal(txHashList1[0], txHashList1[0]))
	require.Equal(t, true, bytes.Equal(txHashList1[1], txHashList1[1]))
	evmAuxStore.ClearData()
}

func TestBloomFilterOperation(t *testing.T) {
	bf1 := []byte("bloomfilter1")
	evmAuxStore, err := LoadStore()
	require.NoError(t, err)
	bf := evmAuxStore.GetBloomFilter(40)
	require.Nil(t, bf)
	db := evmAuxStore.DB()
	tran, err := db.OpenTransaction()
	require.NoError(t, err)
	evmAuxStore.SetBloomFilter(tran, bf1, 30)
	tran.Commit()
	bf = evmAuxStore.GetBloomFilter(30)
	require.Equal(t, true, bytes.Equal(bf, bf1))
	evmAuxStore.ClearData()
}
