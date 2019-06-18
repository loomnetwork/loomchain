package store

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/iavl"
	"github.com/tendermint/tendermint/libs/db"

	"github.com/loomnetwork/loomchain/log"
)

var (
	numBlocks = 10000
	blockSize = 1000

	maxVersions   = 2
	flushInterval = int64(4)
	diskDbType    = "memdb"
	//diskDbType       = "goleveldb"
	blocks []*iavl.Program
	tree   *iavl.MutableTree
)

func TestIavl(t *testing.T) {
	numBlocks = 10
	blockSize = 10
	flushInterval = int64(9)

	log.Setup("debug", "file://-")
	log.Root.With("module", "dual-iavlstore")

	os.RemoveAll("testdata")
	_, err := os.Stat("testdata")
	require.True(t, os.IsNotExist(err))

	blocks = nil
	blocks = iavl.GenerateBlocks2(numBlocks, blockSize)
	tree = iavl.NewMutableTree(db.NewMemDB(), 0)
	for _, program := range blocks {
		require.NoError(t, program.Execute(tree))
	}

	t.Run("testFlush", testFlush)
	t.Run("normal", testNormal)
	t.Run("max versions", testMaxVersions)
}

func testNormal(t *testing.T) {
	diskDb := getDiskDb(t, "testNormal")
	store, err := NewIAVLStore(diskDb, 0, 0, 0)
	require.NoError(t, err)
	executeBlocks(t, blocks, *store)

	diskTree := iavl.NewMutableTree(diskDb, 0)
	_, err = diskTree.Load()
	require.NoError(t, err)
	for _, entry := range store.Range(nil) {
		_, value := tree.Get(entry.Key)
		require.Zero(t, bytes.Compare(value, entry.Value))
		_, diskValue := diskTree.Get(entry.Key)
		require.Zero(t, bytes.Compare(value, diskValue))

	}
	fmt.Println()
	tree.Iterate(func(key []byte, value []byte) bool {
		require.Zero(t, bytes.Compare(value, store.Get(key)))
		_, diskValue := diskTree.Get(key)
		require.Zero(t, bytes.Compare(value, diskValue))
		return false
	})
	diskDb.Close()
}

func testFlush(t *testing.T) {
	diskDb := getDiskDb(t, "testFlush")
	store, err := NewIAVLStore(diskDb, 0, 0, flushInterval)
	require.NoError(t, err)
	executeBlocks(t, blocks, *store)
	_, _, err = store.tree.SaveVersion()

	diskTree := iavl.NewMutableTree(diskDb, 0)
	_, err = diskTree.Load()
	require.NoError(t, err)
	for _, entry := range store.Range(nil) {
		_, value := tree.Get(entry.Key)
		require.Zero(t, bytes.Compare(value, entry.Value))
		_, diskValue := diskTree.Get(entry.Key)
		require.Zero(t, bytes.Compare(value, diskValue))
	}
	tree.Iterate(func(key []byte, value []byte) bool {
		require.Zero(t, bytes.Compare(value, store.Get(key)))
		_, diskValue := diskTree.Get(key)
		require.Zero(t, bytes.Compare(value, diskValue))
		return false
	})
	diskDb.Close()
}

func testMaxVersions(t *testing.T) {
	diskDb := getDiskDb(t, "testMaxVersions")
	store, err := NewIAVLStore(diskDb, int64(maxVersions), 0, 0)
	require.NoError(t, err)
	executeBlocks(t, blocks, *store)

	for i := 1; i <= numBlocks; i++ {
		require.Equal(t,
			i > numBlocks-maxVersions,
			store.tree.VersionExists(int64(i)),
		)
	}
	diskTree := iavl.NewMutableTree(diskDb, 0)
	_, err = diskTree.Load()
	require.NoError(t, err)
	for _, entry := range store.Range(nil) {
		_, value := tree.Get(entry.Key)
		require.Zero(t, bytes.Compare(value, entry.Value))
		_, diskValue := diskTree.Get(entry.Key)
		require.Zero(t, bytes.Compare(value, diskValue))
	}
	tree.Iterate(func(key []byte, value []byte) bool {
		require.Zero(t, bytes.Compare(value, store.Get(key)))
		_, diskValue := diskTree.Get(key)
		require.Zero(t, bytes.Compare(value, diskValue))
		return false
	})
	diskDb.Close()
}

func executeBlocks(t require.TestingT, blocks []*iavl.Program, store IAVLStore) {
	for _, block := range blocks {
		require.NoError(t, block.Execute(store.tree))
		_, _, err := store.SaveVersion()
		require.NoError(t, err)
		require.NoError(t, store.Prune())
	}
}

func getDiskDb(t require.TestingT, name string) db.DB {
	if diskDbType == "goleveldb" {
		diskDb, err := db.NewGoLevelDB(name, "./testdata")
		require.NoError(t, err)
		return diskDb

	} else {
		return db.NewMemDB()
	}
}
