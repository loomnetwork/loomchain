package store

import (
	"bytes"
	"testing"

	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/log"
	"github.com/stretchr/testify/suite"
)

var (
	nonePrefix = []byte("none")
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

func (m *MultiWriterAppStoreTestSuite) TestEnableDisableMultiWriterAppStore() {
	require := m.Require()
	store, err := mockMultiWriterStore(10)
	require.NoError(err)

	// vm keys should be written to both the IAVL & EVM store
	store.Set(evmDBFeatureKey, []byte{})
	store.Set(nonePrefixKey("abcd"), []byte("hello"))
	store.Set(nonePrefixKey("abcde"), []byte("world"))
	store.Set(nonePrefixKey("evmStore"), []byte("yes"))
	store.Set(nonePrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("abcd"), []byte("NewData"))

	rangeData := store.Range(nonePrefix)
	require.Equal(4, len(rangeData))
	require.True(store.Has([]byte("abcd")))

	// vm keys should now only be written to the EVM store
	store.Set(evmDBFeatureKey, []byte{1})
	store.Set(nonePrefixKey("gg"), []byte("world"))
	store.Set(nonePrefixKey("dd"), []byte("yes"))
	store.Set(nonePrefixKey("vv"), []byte("yes"))
	store.Set([]byte("dcba"), []byte("MoreData"))

	rangeData = store.Range(nonePrefix)
	require.Equal(7, len(rangeData))
	require.True(store.Has([]byte("abcd")))
	require.True(store.Has([]byte("dcba")))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreDelete() {
	require := m.Require()
	store, err := mockMultiWriterStore(10)
	require.NoError(err)

	// vm keys should be written to both the IAVL & EVM store
	store.Set(evmDBFeatureKey, []byte{})
	store.Set(nonePrefixKey("abcd"), []byte("hello"))
	store.Set(nonePrefixKey("abcde"), []byte("world"))
	store.Set(nonePrefixKey("evmStore"), []byte("yes"))
	store.Set(nonePrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("vmroot"), []byte("SSSSSSSSSSSSS"))
	store.Set([]byte("abcd"), []byte("NewData"))

	store.Delete(nonePrefixKey("abcd"))
	require.False(store.Has(nonePrefixKey("abcd")))

	rangeData := store.Range(nonePrefix)
	require.Equal(3, len(rangeData))
	require.True(store.Has([]byte("vmroot")))
	require.True(store.Has([]byte("abcd")))

	// vm keys should be written to the EVM store
	store.Set(evmDBFeatureKey, []byte{1})
	rangeData = store.Range(nonePrefix)
	require.Equal(3, len(rangeData))
	require.Equal([]byte("SSSSSSSSSSSSS"), store.Get([]byte("vmroot")))

	store.Set(nonePrefixKey("gg"), []byte("world"))
	store.Set(nonePrefixKey("dd"), []byte("yes"))
	store.Set(nonePrefixKey("vv"), []byte("yes"))
	store.Delete(nonePrefixKey("vv"))
	require.False(store.Has(nonePrefixKey("vv")))

	rangeData = store.Range(nonePrefix)
	require.Equal(5, len(rangeData))
	require.True(store.Has([]byte("abcd")))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSnapShot() {
	require := m.Require()
	store, err := mockMultiWriterStore(10)
	require.NoError(err)

	store.Set(evmDBFeatureKey, []byte{1})
	store.Set(nonePrefixKey("abcd"), []byte("hello"))
	store.Set(nonePrefixKey("abcde"), []byte("world"))
	store.Set(nonePrefixKey("evmStore"), []byte("yes"))
	store.Set(nonePrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("ssssvvv"), []byte("SSSSSSSSSSSSS"))
	store.Set([]byte("abcd"), []byte("NewData"))
	_, _, err = store.SaveVersion()
	require.NoError(err)

	store.Set(nonePrefixKey("abcd"), []byte("hellooooooo"))
	store.Set(nonePrefixKey("abcde"), []byte("vvvvvvvvv"))
	store.Set([]byte("abcd"), []byte("asdfasdf"))

	snapshot := store.GetSnapshot()
	require.Equal([]byte("hello"), snapshot.Get(nonePrefixKey("abcd")))
	require.Equal([]byte("NewData"), snapshot.Get([]byte("abcd")))
	require.Equal([]byte("world"), snapshot.Get(nonePrefixKey("abcde")))

	_, _, err = store.SaveVersion()
	require.NoError(err)

	snapshot = store.GetSnapshot()
	require.Equal([]byte("asdfasdf"), snapshot.Get([]byte("abcd")))
	require.Equal([]byte("hellooooooo"), snapshot.Get(nonePrefixKey("abcd")))
	require.Equal([]byte("vvvvvvvvv"), snapshot.Get(nonePrefixKey("abcde")))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSnapShotFlushInterval() {
	require := m.Require()
	// flush data to disk every 2 blocks
	store, err := mockMultiWriterStore(2)
	require.NoError(err)

	// the first version go to memory
	store.Set([]byte("test1"), []byte("test1"))
	store.Set([]byte("test2"), []byte("test2"))
	_, version, err := store.SaveVersion()
	require.NoError(err)
	require.Equal(int64(1), version)

	store.Set([]byte("test1"), []byte("test1v2"))
	store.Set([]byte("test2"), []byte("test2v2"))

	// this snapshot is from memory
	snapshotv1 := store.GetSnapshot()
	require.Equal([]byte("test1"), snapshotv1.Get([]byte("test1")))
	require.Equal([]byte("test2"), snapshotv1.Get([]byte("test2")))

	// this flushes all data to disk
	_, _, err = store.SaveVersion()
	require.NoError(err)

	// get snapshotv2
	snapshotv2 := store.GetSnapshot()
	require.Equal([]byte("test1v2"), snapshotv2.Get([]byte("test1")))
	require.Equal([]byte("test2v2"), snapshotv2.Get([]byte("test2")))

	// this snapshotv1 should still be accessible
	require.Equal([]byte("test1"), snapshotv1.Get([]byte("test1")))
	require.Equal([]byte("test2"), snapshotv1.Get([]byte("test2")))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSnapShotRange() {
	require := m.Require()
	store, err := mockMultiWriterStore(10)
	require.NoError(err)

	store.Set(evmDBFeatureKey, []byte{1})
	store.Set(nonePrefixKey("abcd"), []byte("hello"))
	store.Set(nonePrefixKey("abcde"), []byte("world"))
	store.Set(nonePrefixKey("evmStore"), []byte("yes"))
	store.Set(nonePrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("ssssvvv"), []byte("SSSSSSSSSSSSS"))
	store.Set([]byte("abcd"), []byte("NewData"))
	store.Set([]byte("uuuu"), []byte("SSSSSSSSSSSSS"))
	store.Set([]byte("sssss"), []byte("NewData"))

	snapshot := store.GetSnapshot()
	rangeData := snapshot.Range(nonePrefix)
	require.Equal(0, len(rangeData))
	_, _, err = store.SaveVersion()
	require.NoError(err)

	snapshot = store.GetSnapshot()
	rangeData = snapshot.Range(nonePrefix)
	require.Equal(4, len(rangeData))
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("abcd")), []byte("hello")))
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("abcde")), []byte("world")))
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("evmStore")), []byte("yes")))
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("aaaa")), []byte("yes")))

	// Modifications shouldn't be visible in the snapshot until the next SaveVersion()
	store.Delete(nonePrefixKey("abcd"))
	store.Delete([]byte("ssssvvv"))

	snapshot = store.GetSnapshot()
	rangeData = snapshot.Range(nonePrefix)
	require.Equal(4, len(rangeData))
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("abcd")), []byte("hello")))
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("abcde")), []byte("world")))
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("evmStore")), []byte("yes")))
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("aaaa")), []byte("yes")))

	_, _, err = store.SaveVersion()
	require.NoError(err)

	snapshot = store.GetSnapshot()
	rangeData = snapshot.Range(nonePrefix)
	require.Equal(3, len(rangeData))
	require.Equal(0, len(snapshot.Get(nonePrefixKey("abcd")))) // has been deleted
	require.Equal(0, len(snapshot.Get([]byte("ssssvvv"))))     // has been deleted
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("abcde")), []byte("world")))
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("evmStore")), []byte("yes")))
	require.Equal(0, bytes.Compare(snapshot.Get(nonePrefixKey("aaaa")), []byte("yes")))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSaveVersion() {
	require := m.Require()
	store, err := mockMultiWriterStore(10)
	require.NoError(err)

	// vm keys should be written to the EVM store
	store.Set(evmDBFeatureKey, []byte{1})
	store.Set(nonePrefixKey("abcd"), []byte("hello"))
	store.Set(nonePrefixKey("abcde"), []byte("world"))
	store.Set(nonePrefixKey("evmStore"), []byte("yes"))
	store.Set(nonePrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("abcd"), []byte("NewData"))
	store.Set([]byte("evmStore"), []byte("iavlStore"))
	store.Set(nonePrefixKey("gg"), []byte("world"))
	store.Set(nonePrefixKey("dd"), []byte("yes"))
	store.Set(nonePrefixKey("vv"), []byte("yes"))

	_, version, err := store.SaveVersion()
	require.Equal(int64(1), version)
	require.NoError(err)

	require.Equal([]byte("hello"), store.Get(nonePrefixKey("abcd")))
	require.Equal([]byte("NewData"), store.Get([]byte("abcd")))
	require.True(store.Has(nonePrefixKey("gg")))
	store.Delete(nonePrefixKey("gg"))

	dataRange := store.Range(nonePrefix)
	require.Equal(6, len(dataRange))
	_, version, err = store.SaveVersion()
	require.Equal(int64(2), version)
	require.NoError(err)

	require.Equal([]byte("hello"), store.Get(nonePrefixKey("abcd")))
	require.Equal([]byte("NewData"), store.Get([]byte("abcd")))
	require.False(store.Has(nonePrefixKey("gg")))
}

func (m *MultiWriterAppStoreTestSuite) TestIAVLRangeWithlimit() {
	require := m.Require()
	store, err := mockMultiWriterStore(10)
	require.NoError(err)

	// write some vm keys to iavl store
	iavlStore := store.appStore
	iavlStore.Set(nonePrefixKey("abcde"), []byte("world"))
	iavlStore.Set(nonePrefixKey("aaaa"), []byte("yes"))
	iavlStore.Set(nonePrefixKey("abcd"), []byte("NewData"))
	iavlStore.Set(nonePrefixKey("evmStore"), []byte("iavlStore"))
	iavlStore.Set(nonePrefixKey("gg"), []byte("world"))
	iavlStore.Set(nonePrefixKey("dd"), []byte("yes"))
	iavlStore.Set(nonePrefixKey("vv"), []byte("yes"))
	_, _, err = store.SaveVersion()
	require.NoError(err)

	// only 4 VM keys will be returned due to the quirkiness of RangeWithLimit
	rangeData := iavlStore.RangeWithLimit(nonePrefix, 5)
	require.Equal(4, len(rangeData))
}

func mockMultiWriterStore(flushInterval int64) (*MultiWriterAppStore, error) {
	memDb, _ := db.LoadMemDB()
	iavlStore, err := NewIAVLStore(memDb, 0, 0, flushInterval)
	if err != nil {
		return nil, err
	}
	memDb, _ = db.LoadMemDB()
	evmStore := NewEvmStore(memDb, 100, 0)
	multiWriterStore, err := NewMultiWriterAppStore(iavlStore, evmStore, false)
	if err != nil {
		return nil, err
	}
	return multiWriterStore, nil
}

func nonePrefixKey(key string) []byte {
	return util.PrefixKey([]byte("none"), []byte(key))
}
