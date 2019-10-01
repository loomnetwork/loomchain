package evmaux

import (
	"fmt"
	"testing"

	"github.com/loomnetwork/go-loom/util"
	"github.com/stretchr/testify/require"
	dbm "github.com/tendermint/tendermint/libs/db"
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
