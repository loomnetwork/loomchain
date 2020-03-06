package store

import (
	"bytes"
	"os"
	"strconv"
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

func TestOrphans(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "dual-iavlstore")

	flushInterval = 5
	memDB := db.NewMemDB()
	baseStore, err := NewIAVLStore(memDB, 0, 0, flushInterval)
	require.NoError(t, err)
	testOrphans(t, baseStore, memDB, flushInterval)

	memDB2 := db.NewMemDB()
	flushStore, err := NewIAVLStore(memDB2, 0, 0, flushInterval)
	require.NoError(t, err)
	testOrphans(t, flushStore, memDB, flushInterval)
}

func testOrphans(t *testing.T, store *IAVLStore, diskDb db.DB, flushInterval int64) {
	store.Set([]byte("k1"), []byte("Fred"))
	store.Set([]byte("k2"), []byte("John"))
	for i := 0; i < int(flushInterval-1); i++ {
		_, _, err := store.SaveVersion(nil)
		require.NoError(t, err)
	}
	store.Set([]byte("k2"), []byte("Bob"))
	store.Set([]byte("k3"), []byte("Harry"))
	_, _, err := store.SaveVersion(nil) // save to disk

	require.NoError(t, err)

	store.Set([]byte("k1"), []byte("Mary"))
	store.Set([]byte("k2"), []byte("Sally"))
	store.Delete([]byte("k3"))
	for i := 0; i < int(flushInterval)-1; i++ {
		_, _, err := store.SaveVersion(nil)
		require.NoError(t, err)
	}

	store.Set([]byte("k2"), []byte("Jim"))
	for i := 0; i < 2*int(flushInterval); i++ {
		_, _, err := store.SaveVersion(nil) // save to disk
		require.NoError(t, err)
	}
	lastVersion := 3 * flushInterval
	store = nil

	newStore, err := NewIAVLStore(diskDb, 0, int64(lastVersion), 0)
	require.NoError(t, err)

	k1Value, _, err := newStore.tree.GetVersionedWithProof([]byte("k1"), flushInterval)
	require.NoError(t, err)
	require.Equal(t, 0, bytes.Compare([]byte("Fred"), k1Value))

	k1Value, _, err = newStore.tree.GetVersionedWithProof([]byte("k1"), 2*flushInterval)
	require.NoError(t, err)
	require.Equal(t, 0, bytes.Compare([]byte("Mary"), k1Value))

	k2Value, _, err := newStore.tree.GetVersionedWithProof([]byte("k2"), flushInterval)
	require.NoError(t, err)
	require.Equal(t, 0, bytes.Compare([]byte("Bob"), k2Value))

	k2Value, _, err = newStore.tree.GetVersionedWithProof([]byte("k2"), 2*flushInterval)
	require.NoError(t, err)
	require.Equal(t, 0, bytes.Compare([]byte("Jim"), k2Value))

	k2Value, _, err = newStore.tree.GetVersionedWithProof([]byte("k2"), 3*flushInterval)
	require.NoError(t, err)
	require.Equal(t, 0, bytes.Compare([]byte("Jim"), k2Value))

	k2Value, _, err = newStore.tree.GetVersionedWithProof([]byte("k3"), flushInterval)
	require.NoError(t, err)
	require.Equal(t, 0, bytes.Compare([]byte("Harry"), k2Value))

	k2Value, _, err = newStore.tree.GetVersionedWithProof([]byte("k3"), 2*flushInterval)
	require.NoError(t, err)
	require.Equal(t, 0, bytes.Compare([]byte(""), k2Value))
}

func TestIavl(t *testing.T) {
	numBlocks = 20
	blockSize = 5
	flushInterval = int64(9)

	log.Setup("debug", "file://-")
	log.Root.With("module", "dual-iavlstore")

	require.NoError(t, os.RemoveAll("testdata"))
	_, err := os.Stat("testdata")
	require.True(t, os.IsNotExist(err))

	blocks = nil
	blocks = iavl.GenerateBlocksHashKeys(numBlocks, blockSize)
	tree = iavl.NewMutableTree(db.NewMemDB(), 0)
	for _, program := range blocks {
		require.NoError(t, program.Execute(tree))
	}

	t.Run("testFlush", testFlush)
	t.Run("normal", testNormal)
	t.Run("max versions", testMaxVersions)
	t.Run("testGetTreeAfterFlush", testGetTreeAfterFlush)
	t.Run("testGetPreviousTree", testGetPreviousTree)
}

func testNormal(t *testing.T) {
	diskDb := getDiskDb(t, "testNormal")
	store, err := NewIAVLStore(diskDb, 0, 0, 0)
	require.NoError(t, err)
	executeBlocks(t, blocks, *store)

	diskTree := iavl.NewMutableTree(diskDb, 0)
	_, err = diskTree.Load()
	require.NoError(t, err)

	tree.Iterate(func(key []byte, value []byte) bool {
		require.Zero(t, bytes.Compare(value, store.Get(key)))
		_, diskValue := diskTree.Get(key)
		require.Zero(t, bytes.Compare(value, diskValue))
		return false
	})
	diskDb.Close()
}

func testGetTreeAfterFlush(t *testing.T) {
	diskDb := getDiskDb(t, "testGetTreeAfterFlush")
	store, err := NewIAVLStore(diskDb, 0, 0, flushInterval)
	require.NoError(t, err)
	var s string
	var flushedVersion, latestVersion int64
	for i := int64(1); i <= flushInterval; i++ {
		s = strconv.FormatInt(i, 10)
		store.Set([]byte("key"+s), []byte("value"+s))

		_, latestVersion, err = store.SaveVersion(nil)

		require.NoError(t, err)
		require.Equal(t, i, latestVersion)

	}
	flushedVersion = latestVersion
	store.Set([]byte("k"), []byte("v"))
	_, latestVersion, err = store.SaveVersion(nil)
	require.NoError(t, err)
	//
	it, err := store.tree.GetImmutable(flushedVersion)
	require.NoError(t, err)
	require.Equal(t, flushedVersion, it.Size())

	// Trying to retrieve a tree on one version after flushed version.
	it, err = store.tree.GetImmutable(latestVersion)
	require.NoError(t, err)
	require.Equal(t, latestVersion, it.Size())

	// Trying to retrieve a tree on one version before flushed version.
	it, err = store.tree.GetImmutable(flushedVersion - 1)
	require.EqualError(t, err, "version does not exist")
	require.Nil(t, it)

	diskDb.Close()
}

func testGetPreviousTree(t *testing.T) {
	diskDb := getDiskDb(t, "testGetPreviousTree")
	store, err := NewIAVLStore(diskDb, 0, 0, flushInterval)
	require.NoError(t, err)

	require.Nil(t, (*iavl.ImmutableTree)(store.previousTree))

	var s string
	var flushedVersion, latestVersion int64
	for i := int64(1); i <= flushInterval; i++ {
		s = strconv.FormatInt(i, 10)
		store.Set([]byte("key"+s), []byte("value"+s))

		_, latestVersion, err = store.SaveVersion(nil)

		require.NoError(t, err)
		require.Equal(t, i, latestVersion)
	}
	flushedVersion = latestVersion

	_, _, err = store.SaveVersion(nil)
	require.NoError(t, err)

	_, _, err = store.SaveVersion(nil)
	require.NoError(t, err)

	// Load IAVLTree of flushed version.
	flushedTree, err := store.tree.GetImmutable(flushedVersion)
	require.NoError(t, err)
	require.Equal(t, flushedVersion, flushedTree.Version())

	require.Equal(t, flushedTree.Version()-1, (*iavl.ImmutableTree)(store.previousTree).Version())
	require.Equal(t, flushedTree.Size()-1, (*iavl.ImmutableTree)(store.previousTree).Size())

	diskDb.Close()
}

func testFlush(t *testing.T) {
	diskDb := getDiskDb(t, "testFlush")
	store, err := NewIAVLStore(diskDb, 0, 0, flushInterval)
	require.NoError(t, err)
	executeBlocks(t, blocks, *store)
	_, _, err = store.tree.SaveVersion()
	require.NoError(t, err)
	diskTree := iavl.NewMutableTree(diskDb, 0)
	_, err = diskTree.Load()
	require.NoError(t, err)

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
		_, _, err := store.SaveVersion(nil)
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
