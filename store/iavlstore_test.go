package store

import (
	"bytes"
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

	maxVersions      = 2
	diskDbType       = "memdb"
	saveFrequency    = 7
	versionFrequency = 5
	testMinCache     = uint64(10)
	testMaxCache     = uint64(500)
	verbose          = true
	blocks           []*iavl.Program
	tree             *iavl.MutableTree
)

func TestIavl(t *testing.T) {
	numBlocks = 10
	blockSize = 10

	log.Setup("debug", "file://-")
	log.Root.With("module", "dual-iavlstore")

	require.NoError(t, os.RemoveAll("testdata"))
	_, err := os.Stat("testdata")
	require.True(t, os.IsNotExist(err))

	blocks = nil
	blocks = iavl.GenerateBlocks2(numBlocks, blockSize)
	tree = iavl.NewMutableTree(db.NewMemDB(), 0)
	for _, program := range blocks {
		require.NoError(t, program.Execute(tree))
		_, _, err := tree.SaveVersion()
		require.NoError(t, err)
	}

	t.Run("normal", testNormal)
	t.Run("varable cache", testVariableCache)
	t.Run("max versions & max frequency", testMaxVersionFrequency)
	t.Run("max versions", testMaxVersions)
	t.Run("save frequency", testSaveFrequency) // add two to save frequency?!
	t.Run("max versions, max versions & save frequency", testMaxVersionFrequencySaveFrequency)
}

func testNormal(t *testing.T) {
	diskDb := getDiskDb(t, "testNormal")
	store, err := NewDiffIavlStore(diskDb, 0, 0, 0, 0, 0, 0)
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
	store, err := NewDiffIavlStore(diskDb, int64(maxVersions), 0, 0, 0, 0, 0)
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

func testMaxVersionFrequency(t *testing.T) {
	diskDb := getDiskDb(t, "testMaxVersionFrequency")
	store, err := NewDiffIavlStore(diskDb, int64(maxVersions), 0, 0, uint64(versionFrequency), 0, 0)
	require.NoError(t, err)
	executeBlocks(t, blocks, *store)

	for i := 1; i <= numBlocks; i++ {
		require.Equal(t,
			i > numBlocks-maxVersions || i%versionFrequency == 0,
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

func testSaveFrequency(t *testing.T) {
	diskDb := getDiskDb(t, "testSaveFrequency")
	store, err := NewDiffIavlStore(diskDb, 0, 0, uint64(saveFrequency), 0, 0, 0)
	require.NoError(t, err)
	executeBlocks(t, blocks, *store)

	diskTree := iavl.NewMutableTree(diskDb, 0)
	_, err = diskTree.Load()
	require.NoError(t, err)

	for i := 1; i <= numBlocks; i++ {
		if i/saveFrequency < numBlocks/saveFrequency || i%saveFrequency == 0 {
			require.True(t, diskTree.VersionExists(int64(i)))
			iDiskTree, err := diskTree.GetImmutable(int64(i))
			require.NoError(t, err)
			itree, err := tree.GetImmutable(int64(i))
			require.NoError(t, err)
			itree.Iterate(func(key []byte, value []byte) bool {
				_, diskValue := iDiskTree.Get(key)
				require.Zero(t, bytes.Compare(value, diskValue))
				return false
			})
		} else {
			require.False(t, diskTree.VersionExists(int64(i)))
		}
	}
	for _, entry := range store.Range(nil) {
		_, value := tree.Get(entry.Key)
		require.Zero(t, bytes.Compare(value, entry.Value))
	}
	tree.Iterate(func(key []byte, value []byte) bool {
		require.Zero(t, bytes.Compare(value, store.Get(key)))
		return false
	})

	diskDb.Close()
}

func testVariableCache(t *testing.T) {
	diskDb := getDiskDb(t, "testNormal")
	store, err := NewDiffIavlStore(diskDb, 0, 0, 0, 0, testMinCache, testMaxCache)
	require.NoError(t, err)
	executeBlocks(t, blocks, *store)

	diskTree := iavl.NewMutableTree(diskDb, 0)
	_, err = diskTree.Load()
	require.NoError(t, err)
	for _, entry := range store.Range(nil) {
		_, value := tree.Get(entry.Key)
		require.Zero(t, bytes.Compare(value, entry.Value))
	}
	tree.Iterate(func(key []byte, value []byte) bool {
		require.Zero(t, bytes.Compare(value, store.Get(key)))
		return false
	})
	diskDb.Close()
}

func testMaxVersionFrequencySaveFrequency(t *testing.T) {
	diskDb := getDiskDb(t, "testMaxVersionFrequencySaveFrequency")
	store, err := NewDiffIavlStore(diskDb, int64(maxVersions), 0, uint64(saveFrequency), uint64(versionFrequency), 0, 0)
	require.NoError(t, err)
	executeBlocks(t, blocks, *store)

	diskTree := iavl.NewMutableTree(diskDb, 0)
	_, err = diskTree.Load()
	require.NoError(t, err)
	for i := 1; i <= numBlocks; i++ {
		lastSave := (numBlocks / saveFrequency) * saveFrequency
		if i <= lastSave {
			require.Equal(t,
				(i%versionFrequency == 0 || i > lastSave-maxVersions-1),
				diskTree.VersionExists(int64(i)),
			)
		} else {
			require.False(t, diskTree.VersionExists(int64(i)))
		}
	}
	for _, entry := range store.Range(nil) {
		_, value := tree.Get(entry.Key)
		require.Zero(t, bytes.Compare(value, entry.Value))
	}
	tree.Iterate(func(key []byte, value []byte) bool {
		require.Zero(t, bytes.Compare(value, store.Get(key)))
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

func generateBlocks(t require.TestingT) {
	blocks = nil
	blocks = iavl.GenerateBlocks(numBlocks, blockSize)
	tree = iavl.NewMutableTree(db.NewMemDB(), 0)
	for _, program := range blocks {
		require.NoError(t, program.Execute(tree))
	}
}

func generateBlocks2() error {
	blocks = nil
	blocks = iavl.GenerateBlocks(numBlocks, blockSize)
	tree = iavl.NewMutableTree(db.NewMemDB(), 0)
	for _, program := range blocks {
		err := program.Execute(tree)
		if err != nil {
			return err
		}
	}
	return nil
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
