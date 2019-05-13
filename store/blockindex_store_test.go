package store

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
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
	memoryStore := NewMemoryBlockIndexStore()
	testBlockIndexStore(t, memoryStore)

	_ = os.RemoveAll(LevelDBFilename)
	_, err := os.Stat(LevelDBFilename)
	require.True(t, os.IsNotExist(err))
	levelDbStore, err := NewLevelDBBlockIndexStore()
	require.NoError(t, err)
	testBlockIndexStore(t, levelDbStore)
}

func testBlockIndexStore(t *testing.T, bs BlockIndexStore) {
	for _, hashHeight := range hashHeights {
		require.NoError(t, bs.SetBlockHashAtHeight(hashHeight.hash, hashHeight.height))
	}
	for _, hashHeight := range hashHeights {
		height, err := bs.GetBlockHeightByHash(hashHeight.hash)
		require.NoError(t, err)
		require.Equal(t, hashHeight.height, height)
	}
	_, err := bs.GetBlockHeightByHash([]byte("Non existent block-hash"))
	require.Error(t, err)
	require.NoError(t, bs.Close())
}
