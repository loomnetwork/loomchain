package store

import (
	"strconv"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom/config"
	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/log"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
)

type MultiWriterAppStoreTestSuite struct {
	suite.Suite
}

func (m *MultiWriterAppStoreTestSuite) SetupTest() {
	log.Setup("debug", "file://-")
}

func TestMultiWriterAppStoreTestSuite(t *testing.T) {
	suite.Run(t, new(MultiWriterAppStoreTestSuite))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSnapshotFlushInterval() {
	require := m.Require()
	// flush data to disk every 2 blocks
	store, err := mockMultiWriterStore(2, 2)
	require.NoError(err)

	// the first version go to memory
	store.Set([]byte("test1"), []byte("test1"))
	store.Set([]byte("test2"), []byte("test2"))
	_, version, err := store.SaveVersion(nil)
	require.NoError(err)
	require.Equal(int64(1), version)

	store.Set([]byte("test1"), []byte("test1v2"))
	store.Set([]byte("test2"), []byte("test2v2"))

	// this snapshot is from memory
	snapshotv1, err := store.GetSnapshotAt(0)
	require.NoError(err)
	require.Equal([]byte("test1"), snapshotv1.Get([]byte("test1")))
	require.Equal([]byte("test2"), snapshotv1.Get([]byte("test2")))

	// this flushes all data to disk
	_, _, err = store.SaveVersion(nil)
	require.NoError(err)

	// get snapshotv2
	snapshotv2, err := store.GetSnapshotAt(0)
	require.NoError(err)
	require.Equal([]byte("test1v2"), snapshotv2.Get([]byte("test1")))
	require.Equal([]byte("test2v2"), snapshotv2.Get([]byte("test2")))

	// this snapshotv1 should still be accessible
	require.Equal([]byte("test1"), snapshotv1.Get([]byte("test1")))
	require.Equal([]byte("test2"), snapshotv1.Get([]byte("test2")))

}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreGetSnapshotAtFlushInterval() {
	require := m.Require()
	// flush data to disk every 2 blocks
	store, err := mockMultiWriterStore(2, 2)
	require.NoError(err)

	// the first version will be in-memory only
	store.Set([]byte("test1"), []byte("test1"))
	store.Set([]byte("test2"), []byte("test2"))
	_, version, err := store.SaveVersion(nil)
	require.NoError(err)
	require.Equal(int64(1), version)

	store.Set([]byte("test1"), []byte("test1v2"))
	store.Set([]byte("test2"), []byte("test2v2"))

	// this snapshot is from memory
	snapshotv1, err := store.GetSnapshotAt(0)
	require.NoError(err)
	require.Equal([]byte("test1"), snapshotv1.Get([]byte("test1")))
	require.Equal([]byte("test2"), snapshotv1.Get([]byte("test2")))

	// this flushes all data to disk
	_, _, err = store.SaveVersion(nil)
	require.NoError(err)

	// get snapshotv2
	snapshotv2, err := store.GetSnapshotAt(0)
	require.NoError(err)
	require.Equal([]byte("test1v2"), snapshotv2.Get([]byte("test1")))
	require.Equal([]byte("test2v2"), snapshotv2.Get([]byte("test2")))

	// this snapshotv1 should still be accessible
	require.Equal([]byte("test1"), snapshotv1.Get([]byte("test1")))
	require.Equal([]byte("test2"), snapshotv1.Get([]byte("test2")))
}

// This test checks that GetSnapshotAt() can be used to access the store version preceeding the
// one that's flushed to disk.
func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreGetSnapshotAtPreviousTree() {
	require := m.Require()
	var flushInterval int64 = 5
	// flush data to disk every 5 blocks
	store, err := mockMultiWriterStore(flushInterval, flushInterval)
	require.NoError(err)

	require.Nil((*IAVLStore)(store.appStore.previousTree))

	// Set a key and value and save for 5 versions
	var s string
	var latestVersion int64
	for i := int64(1); i <= flushInterval; i++ {
		s = strconv.FormatInt(i, 10)
		store.Set(vmPrefixKey(s), []byte("value"+s))
		_, latestVersion, err = store.SaveVersion(nil)

		require.NoError(err)
		require.Equal(i, latestVersion)
	}

	// The snapshot for the last in-mem-only version should still be available
	_, err = store.GetSnapshotAt(latestVersion - 1)
	require.NoError(err)

	// Check that the snapshot for the latest version contains all the previously written keys.
	snap, err := store.GetSnapshotAt(0)
	require.NoError(err)
	require.Equal(latestVersion, int64(len(snap.Range(vmPrefix))))

	// Set another 4 unique keys
	for i := int64(6); i <= int64(9); i++ {
		s = strconv.FormatInt(i, 10)
		store.Set(vmPrefixKey(s), []byte("value"+s))
		_, latestVersion, err = store.SaveVersion(nil)

		require.NoError(err)
		require.Equal(i, latestVersion)
	}

	// Since the last version that was flushed to disk is 5 the snapshot for version 4 should still
	// be available
	_, err = store.GetSnapshotAt(4)
	require.NoError(err)

	// Save another version and flush it to disk
	store.Set(vmPrefixKey("ten"), []byte("value10"))
	_, latestVersion, err = store.SaveVersion(nil)
	require.NoError(err)
	require.Equal(int64(10), latestVersion)

	// Now that version 10 has been flushed to disk the snapshot for version 4 should no longer be
	// available
	_, err = store.GetSnapshotAt(4)
	require.EqualError(err, "failed to load immutable tree for version 4: version does not exist")
	require.Error(err)

	// The snapshot for the last in-mem-only version should still be available
	snap, err = store.GetSnapshotAt(latestVersion - 1)
	require.NoError(err)
	require.Equal(int64(9), int64(len(snap.Range(vmPrefix))))

	snap, err = store.GetSnapshotAt(latestVersion)
	require.NoError(err)
	require.Equal([]byte("value10"), snap.Get(vmPrefixKey("ten")))
	require.Equal(int64(10), int64(len(snap.Range(vmPrefix))))

}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterEvmStoreGetSnapshot() {
	require := m.Require()
	var flushInterval int64 = 5
	var numCachedRoots int = 5

	memDb, _ := db.LoadMemDB()
	iavlStore, err := NewIAVLStore(memDb, 0, 0, flushInterval)
	require.NoError(err)

	// flush data to disk every 5 block and set cache size to 5.
	memDb, _ = db.LoadMemDB()
	evmStore := NewEvmStore(memDb, numCachedRoots, flushInterval)
	store, err := NewMultiWriterAppStore(iavlStore, evmStore)
	require.NoError(err)

	var latestVersion, version int64
	var root []byte
	for i := int64(1); i <= int64(20); i++ {
		s := strconv.FormatInt(i, 10)
		store.evmStore.SetCurrentRoot([]byte(s))
		_, version, err := store.SaveVersion(nil)
		require.NoError(err)
		latestVersion = version
	}

	// All flushed version should be available.
	var flushedVersion = []int64{5, 10, 15, 20}
	for _, fv := range flushedVersion {
		root, version = store.evmStore.GetRootAt(fv)
		require.Equal(fv, version)
		require.True(len(root) > 0)
	}

	// Try to get 2 version ahead of latest version.
	// Should return latest version.
	root, version = store.evmStore.GetRootAt(latestVersion + 2)
	require.Equal(latestVersion, version)
	require.True(len(root) > 0)

	// Since we set numCachedRoots to 5
	// 5 version before latest version(20) should  still be available
	var recentFive = []int64{16, 17, 18, 19}
	for _, r := range recentFive {
		root, version = store.evmStore.GetRootAt(r)
		require.Equal(r, version)
		require.True(len(root) > 0)
	}

	// Try to load unavailable version.
	// Should return most recent version that less than target version.
	root, version = store.evmStore.GetRootAt(13)
	require.Equal(int64(10), version)
	require.True(len(root) > 0)

	root, version = store.evmStore.GetRootAt(9)
	require.Equal(int64(5), version)
	require.True(len(root) > 0)

	root, version = store.evmStore.GetRootAt(4)
	require.Equal(int64(0), version)
	require.Nil(root)
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSaveVersion() {
	require := m.Require()
	store, err := mockMultiWriterStore(10, -1)
	require.NoError(err)

	// all keys (including vm keys) should be written to the IAVL store
	store.Set(vmPrefixKey("abcd"), []byte("hello"))
	store.Set(vmPrefixKey("abcde"), []byte("world"))
	store.Set(vmPrefixKey("evmStore"), []byte("yes"))
	store.Set(vmPrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("abcd"), []byte("NewData"))
	store.Set([]byte("evmStore"), []byte("iavlStore"))
	store.Set(vmPrefixKey("gg"), []byte("world"))
	store.Set(vmPrefixKey("dd"), []byte("yes"))
	store.Set(vmPrefixKey("vv"), []byte("yes"))

	_, version, err := store.SaveVersion(nil)
	require.Equal(int64(1), version)
	require.NoError(err)

	require.Equal([]byte("hello"), store.Get(vmPrefixKey("abcd")))
	require.Equal([]byte("NewData"), store.Get([]byte("abcd")))
	require.True(store.Has(vmPrefixKey("gg")))
	store.Delete(vmPrefixKey("gg"))

	dataRange := store.Range(vmPrefix)
	require.Equal(6, len(dataRange))

	_, version, err = store.SaveVersion(nil)
	require.Equal(int64(2), version)
	require.NoError(err)

	require.Equal([]byte("hello"), store.Get(vmPrefixKey("abcd")))
	require.Equal([]byte("NewData"), store.Get([]byte("abcd")))
	require.False(store.Has(vmPrefixKey("gg")))
}

func (m *MultiWriterAppStoreTestSuite) TestPruningEvmKeys() {
	require := m.Require()
	store, err := mockMultiWriterStore(10, 10)
	require.NoError(err)

	// write some vm keys to iavl store
	iavlStore := store.appStore
	iavlStore.Set(vmPrefixKey("abcde"), []byte("world"))
	iavlStore.Set(vmPrefixKey("aaaa"), []byte("yes"))
	iavlStore.Set(vmPrefixKey("abcd"), []byte("NewData"))
	iavlStore.Set(vmPrefixKey("evmStore"), []byte("iavlStore"))
	iavlStore.Set(vmPrefixKey("gg"), []byte("world"))
	iavlStore.Set(vmPrefixKey("dd"), []byte("yes"))
	iavlStore.Set(vmPrefixKey("vv"), []byte("yes"))
	_, version, err := store.SaveVersion(nil)
	require.NoError(err)
	require.Equal(int64(1), version)
	require.Equal(version, iavlStore.Version())
	_, evmStoreVer := store.evmStore.Version()
	require.Equal(version, evmStoreVer)

	newStore, err := NewMultiWriterAppStore(iavlStore, store.evmStore)
	require.NoError(err)

	rangeData := iavlStore.Range([]byte("vm"))
	require.Equal(7, len(rangeData))

	// prune max 5 vm keys per block
	cfg := config.DefaultConfig()
	cfg.AppStore.NumEvmKeysToPrune = 5
	configBytes, err := proto.Marshal(cfg)
	require.NoError(err)
	newStore.Set([]byte(configKey), configBytes)

	// prune VM keys
	// NOTE: only 3 vm keys will actually get pruned due to the quirkiness of RangeWithLimit
	_, version, err = newStore.SaveVersion(nil)
	require.Equal(int64(2), version)
	require.NoError(err)

	// expect number of vm keys to be 7-3 = 4
	rangeData = iavlStore.Range([]byte("vm"))
	require.Equal(4, len(rangeData))

	// prune VM keys
	// NOTE: once again only 3 vm keys will get pruned
	_, version, err = newStore.SaveVersion(nil)
	require.Equal(int64(3), version)
	require.NoError(err)

	// expect number of vm keys to be 4-3 = 1
	rangeData = iavlStore.Range([]byte("vm"))
	require.Equal(1, len(rangeData))

	// prune VM keys
	_, version, err = newStore.SaveVersion(nil)
	require.Equal(int64(4), version)
	require.NoError(err)

	// all the VM keys should be gone now
	rangeData = iavlStore.Range([]byte("vm"))
	require.Equal(0, len(rangeData))
}

func (m *MultiWriterAppStoreTestSuite) TestIAVLRangeWithlimit() {
	require := m.Require()
	store, err := mockMultiWriterStore(10, 10)
	require.NoError(err)

	// write some vm keys to iavl store
	iavlStore := store.appStore
	iavlStore.Set(vmPrefixKey("abcde"), []byte("world"))
	iavlStore.Set(vmPrefixKey("aaaa"), []byte("yes"))
	iavlStore.Set(vmPrefixKey("abcd"), []byte("NewData"))
	iavlStore.Set(vmPrefixKey("evmStore"), []byte("iavlStore"))
	iavlStore.Set(vmPrefixKey("gg"), []byte("world"))
	iavlStore.Set(vmPrefixKey("dd"), []byte("yes"))
	iavlStore.Set(vmPrefixKey("vv"), []byte("yes"))
	_, _, err = store.SaveVersion(nil)
	require.NoError(err)

	// only 4 VM keys will be returned due to the quirkiness of RangeWithLimit
	rangeData := iavlStore.RangeWithLimit([]byte("vm"), 5)
	require.Equal(4, len(rangeData))
}

func (m *MultiWriterAppStoreTestSuite) TestStoreRange() {
	require := m.Require()
	mws, err := mockMultiWriterStore(0, 0)
	require.NoError(err)
	prefixes, entries := populateStore(mws)
	verifyRange(require, "MultiWriterAppStore", mws, prefixes, entries)
	_, _, err = mws.SaveVersion(nil)
	require.NoError(err)
	verifyRange(require, "MultiWriterAppStore", mws, prefixes, entries)
}

func (m *MultiWriterAppStoreTestSuite) TestSnapshotRange() {
	require := m.Require()
	mws, err := mockMultiWriterStore(0, 0)
	require.NoError(err)
	prefixes, entries := populateStore(mws)
	verifyRange(require, "MultiWriterAppStore", mws, prefixes, entries)
	mws.SaveVersion(nil)

	// snapshot should see all the data that was saved to disk
	func() {
		snap, err := mws.GetSnapshotAt(0)
		require.NoError(err)
		defer snap.Release()

		verifyRange(require, "MultiWriterAppStoreSnapshot", snap, prefixes, entries)
	}()
}

func (m *MultiWriterAppStoreTestSuite) TestConcurrentSnapshots() {
	require := m.Require()
	mws, err := mockMultiWriterStore(0, 0)
	require.NoError(err)
	verifyConcurrentSnapshots(require, mws)
}

func mockMultiWriterStore(appStoreFlushInterval, evmStoreFlushInterval int64) (*MultiWriterAppStore, error) {
	// Using different flush intervals for the app & evm stores is not supported.
	if appStoreFlushInterval > 0 && evmStoreFlushInterval > 0 && appStoreFlushInterval != evmStoreFlushInterval {
		return nil, errors.New("positive flush intervals must be consistent")
	}

	memDb, _ := db.LoadMemDB()
	iavlStore, err := NewIAVLStore(memDb, 0, 0, appStoreFlushInterval)
	if err != nil {
		return nil, err
	}
	memDb, _ = db.LoadMemDB()
	evmStore := NewEvmStore(memDb, 100, evmStoreFlushInterval)
	multiWriterStore, err := NewMultiWriterAppStore(iavlStore, evmStore)
	if err != nil {
		return nil, err
	}
	return multiWriterStore, nil
}

func vmPrefixKey(key string) []byte {
	return util.PrefixKey([]byte("vm"), []byte(key))
}
