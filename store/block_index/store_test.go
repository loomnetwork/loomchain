package blockindex

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/loomnetwork/loomchain/db"
)

var (
	hashHeights = []struct {
		hash   []byte
		height uint64
	}{
		{[]byte("1"), 1},
		{[]byte("two"), 2},
		{[]byte("7432fe49764bc0dea050ac4929d9c89d4b5bac841637a42e0fa8e0f8a608aff3"), 3},
		{[]byte("889b48bf9c65dd8ae8fa3b86b755318bb91c10483111cf0a1444afacd3dc313b"), 4},
	}
)

func TestBlockIndexStore(t *testing.T) {
	memoryStore, err := NewBlockIndexStore(db.MemDBackend, LevelDBFilename, ".", 0, 0, false)
	require.NoError(t, err)
	testBlockIndexStore(t, memoryStore)

	_ = os.RemoveAll(LevelDBFilename)
	_, err = os.Stat(LevelDBFilename)
	require.True(t, os.IsNotExist(err))
	golevelDbStore, err := NewBlockIndexStore(db.GoLevelDBBackend, LevelDBFilename, ".", 0, 0, false)
	require.NoError(t, err)
	testBlockIndexStore(t, golevelDbStore)
}

func testBlockIndexStore(t *testing.T, bs BlockIndexStore) {
	for _, hashHeight := range hashHeights {
		bs.SetBlockHashAtHeight(hashHeight.height, hashHeight.hash)
	}
	for _, hashHeight := range hashHeights {
		height, err := bs.GetBlockHeightByHash(hashHeight.hash)
		require.NoError(t, err)
		require.Equal(t, hashHeight.height, height)
	}
	_, err := bs.GetBlockHeightByHash([]byte("Non existent block-hash"))
	require.Equal(t, ErrNotFound, err)
	bs.Close()
}
