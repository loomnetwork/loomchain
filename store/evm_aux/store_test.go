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

	require.NoError(t, err)
	// add dup EVM txhash keys prefixed with dtx
	for i := 0; i < 100; i++ {
		evmAuxStore.db.Put(dupTxHashKey([]byte(fmt.Sprintf("hash:%d", i))), []byte{1}, nil)
	}
	// add 100 keys prefixed with hash
	for i := 0; i < 100; i++ {
		evmAuxStore.db.Put([]byte(fmt.Sprintf("hash:%d", i)), []byte{1}, nil)
	}
	// add another 100 keys prefixed with ahash
	for i := 0; i < 100; i++ {
		evmAuxStore.db.Put([]byte(fmt.Sprintf("ahash:%d", i)), []byte{1}, nil)
	}
	require.NoError(t, evmAuxStore.Close())

	evmAuxStore2, err := LoadStore()
	require.NoError(t, err)
	dupEvmTxHashes := evmAuxStore2.GetDupEVMTxHashes()
	require.Equal(t, 100, len(dupEvmTxHashes))
	evmAuxStore2.Close()
	evmAuxStore2.ClearData()
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
