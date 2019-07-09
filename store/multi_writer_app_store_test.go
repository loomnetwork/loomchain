package store

import (
	"bytes"
	"testing"

	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/stretchr/testify/suite"
)

type MultiWriterAppStoreTestSuite struct {
	suite.Suite
}

func (m *MultiWriterAppStoreTestSuite) SetupTest() {
}

func TestMultiWriterAppStoreTestSuite(t *testing.T) {
	suite.Run(t, new(MultiWriterAppStoreTestSuite))
}

func (m *MultiWriterAppStoreTestSuite) TestEnableDisableMultiWriterAppStore() {
	require := m.Require()
	store, err := mockMultiWriterStore()
	require.NoError(err)

	// vm keys should be written to both the IAVL & EVM store
	store.Set(evmDBFeatureKey, []byte{})
	store.Set(vmPrefixKey("abcd"), []byte("hello"))
	store.Set(vmPrefixKey("abcde"), []byte("world"))
	store.Set(vmPrefixKey("evmStore"), []byte("yes"))
	store.Set(vmPrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("abcd"), []byte("NewData"))

	rangeData := store.Range(vmPrefix)
	require.Equal(4, len(rangeData))
	require.True(store.Has([]byte("abcd")))

	// vm keys should now only be written to the EVM store
	store.Set(evmDBFeatureKey, []byte{1})
	store.Set(vmPrefixKey("gg"), []byte("world"))
	store.Set(vmPrefixKey("dd"), []byte("yes"))
	store.Set(vmPrefixKey("vv"), []byte("yes"))
	store.Set([]byte("dcba"), []byte("MoreData"))

	rangeData = store.Range(vmPrefix)
	require.Equal(7, len(rangeData))
	require.True(store.Has([]byte("abcd")))
	require.True(store.Has([]byte("dcba")))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreDelete() {
	require := m.Require()
	store, err := mockMultiWriterStore()
	require.NoError(err)

	// vm keys should be written to both the IAVL & EVM store
	store.Set(evmDBFeatureKey, []byte{})
	store.Set(vmPrefixKey("abcd"), []byte("hello"))
	store.Set(vmPrefixKey("abcde"), []byte("world"))
	store.Set(vmPrefixKey("evmStore"), []byte("yes"))
	store.Set(vmPrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("vmroot"), []byte("SSSSSSSSSSSSS"))
	store.Set([]byte("abcd"), []byte("NewData"))

	store.Delete(vmPrefixKey("abcd"))
	require.False(store.Has(vmPrefixKey("abcd")))

	rangeData := store.Range(vmPrefix)
	require.Equal(3, len(rangeData))
	require.True(store.Has([]byte("vmroot")))
	require.True(store.Has([]byte("abcd")))

	// vm keys should be written to the EVM store
	store.Set(evmDBFeatureKey, []byte{1})
	rangeData = store.Range(vmPrefix)
	require.Equal(3, len(rangeData))
	require.Equal([]byte("SSSSSSSSSSSSS"), store.Get([]byte("vmroot")))

	store.Set(vmPrefixKey("gg"), []byte("world"))
	store.Set(vmPrefixKey("dd"), []byte("yes"))
	store.Set(vmPrefixKey("vv"), []byte("yes"))
	store.Delete(vmPrefixKey("vv"))
	require.False(store.Has(vmPrefixKey("vv")))

	rangeData = store.Range(vmPrefix)
	require.Equal(5, len(rangeData))
	require.True(store.Has([]byte("abcd")))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSnapShot() {
	require := m.Require()
	store, err := mockMultiWriterStore()
	require.NoError(err)

	store.Set(evmDBFeatureKey, []byte{1})
	store.Set(vmPrefixKey("abcd"), []byte("hello"))
	store.Set(vmPrefixKey("abcde"), []byte("world"))
	store.Set(vmPrefixKey("evmStore"), []byte("yes"))
	store.Set(vmPrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("ssssvvv"), []byte("SSSSSSSSSSSSS"))
	store.Set([]byte("abcd"), []byte("NewData"))
	_, _, err = store.SaveVersion()
	require.NoError(err)

	store.Set(vmPrefixKey("abcd"), []byte("hellooooooo"))
	store.Set(vmPrefixKey("abcde"), []byte("vvvvvvvvv"))
	store.Set([]byte("abcd"), []byte("asdfasdf"))

	snapshot := store.GetSnapshot()
	require.Equal([]byte("hello"), snapshot.Get(vmPrefixKey("abcd")))
	require.Equal([]byte("NewData"), snapshot.Get([]byte("abcd")))
	require.Equal([]byte("world"), snapshot.Get(vmPrefixKey("abcde")))

	_, _, err = store.SaveVersion()
	require.NoError(err)

	snapshot = store.GetSnapshot()
	require.Equal([]byte("asdfasdf"), snapshot.Get([]byte("abcd")))
	require.Equal([]byte("hellooooooo"), snapshot.Get(vmPrefixKey("abcd")))
	require.Equal([]byte("vvvvvvvvv"), snapshot.Get(vmPrefixKey("abcde")))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSnapShotRange() {
	require := m.Require()
	store, err := mockMultiWriterStore()
	require.NoError(err)

	store.Set(evmDBFeatureKey, []byte{1})
	store.Set(vmPrefixKey("abcd"), []byte("hello"))
	store.Set(vmPrefixKey("abcde"), []byte("world"))
	store.Set(vmPrefixKey("evmStore"), []byte("yes"))
	store.Set(vmPrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("ssssvvv"), []byte("SSSSSSSSSSSSS"))
	store.Set([]byte("abcd"), []byte("NewData"))
	store.Set([]byte("uuuu"), []byte("SSSSSSSSSSSSS"))
	store.Set([]byte("sssss"), []byte("NewData"))

	snapshot := store.GetSnapshot()
	rangeData := snapshot.Range(vmPrefix)
	require.Equal(0, len(rangeData))
	_, _, err = store.SaveVersion()
	require.NoError(err)

	snapshot = store.GetSnapshot()
	rangeData = snapshot.Range(vmPrefix)
	require.Equal(4+1, len(rangeData)) // +1 for evm root stored by EVM store
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("abcd")), []byte("hello")))
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("abcde")), []byte("world")))
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("evmStore")), []byte("yes")))
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("aaaa")), []byte("yes")))

	// Modifications shouldn't be visible in the snapshot until the next SaveVersion()
	store.Delete(vmPrefixKey("abcd"))
	store.Delete([]byte("ssssvvv"))

	snapshot = store.GetSnapshot()
	rangeData = snapshot.Range(vmPrefix)
	require.Equal(4+1, len(rangeData)) // +1 for evm root stored by EVM store
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("abcd")), []byte("hello")))
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("abcde")), []byte("world")))
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("evmStore")), []byte("yes")))
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("aaaa")), []byte("yes")))

	_, _, err = store.SaveVersion()
	require.NoError(err)

	snapshot = store.GetSnapshot()
	rangeData = snapshot.Range(vmPrefix)
	require.Equal(3+1, len(rangeData))                       // +1 for evm root stored by EVM store
	require.Equal(0, len(snapshot.Get(vmPrefixKey("abcd")))) // has been deleted
	require.Equal(0, len(snapshot.Get([]byte("ssssvvv"))))   // has been deleted
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("abcde")), []byte("world")))
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("evmStore")), []byte("yes")))
	require.Equal(0, bytes.Compare(snapshot.Get(vmPrefixKey("aaaa")), []byte("yes")))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSaveVersion() {
	require := m.Require()
	store, err := mockMultiWriterStore()
	require.NoError(err)

	// vm keys should be written to the EVM store
	store.Set(evmDBFeatureKey, []byte{1})
	store.Set(vmPrefixKey("abcd"), []byte("hello"))
	store.Set(vmPrefixKey("abcde"), []byte("world"))
	store.Set(vmPrefixKey("evmStore"), []byte("yes"))
	store.Set(vmPrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("abcd"), []byte("NewData"))
	store.Set([]byte("evmStore"), []byte("iavlStore"))
	store.Set(vmPrefixKey("gg"), []byte("world"))
	store.Set(vmPrefixKey("dd"), []byte("yes"))
	store.Set(vmPrefixKey("vv"), []byte("yes"))

	_, version, err := store.SaveVersion()
	require.Equal(int64(1), version)
	require.NoError(err)

	require.Equal([]byte("hello"), store.Get(vmPrefixKey("abcd")))
	require.Equal([]byte("NewData"), store.Get([]byte("abcd")))
	require.True(store.Has(vmPrefixKey("gg")))
	store.Delete(vmPrefixKey("gg"))

	dataRange := store.Range(vmPrefix)
	require.Equal(6+1, len(dataRange)) // +1 is for the evm root that written by the EVM store itself

	_, version, err = store.SaveVersion()
	require.Equal(int64(2), version)
	require.NoError(err)

	require.Equal([]byte("hello"), store.Get(vmPrefixKey("abcd")))
	require.Equal([]byte("NewData"), store.Get([]byte("abcd")))
	require.False(store.Has(vmPrefixKey("gg")))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterStoreCacheRange() {
	require := m.Require()
	store, err := mockMultiWriterStore()
	require.NoError(err)

	store.Set(featureKey("db:evm"), []byte("true"))
	store.Set(featureKey("chainconfig:v1.1"), []byte("true"))
	store.Set(featureKey("dpos:v3.1"), []byte("true"))

	rangeData := store.Range(featurePrefix)
	rangeDataAppStore := store.appStore.Range(featurePrefix)
	require.Equal(rangeData[0].Key, rangeDataAppStore[0].Key)
	require.Equal(rangeData[1].Key, rangeDataAppStore[1].Key)
	require.Equal(rangeData[2].Key, rangeDataAppStore[2].Key)

	require.Equal(rangeData[0].Value, rangeDataAppStore[0].Value)
	require.Equal(rangeData[1].Value, rangeDataAppStore[1].Value)
	require.Equal(rangeData[2].Value, rangeDataAppStore[2].Value)
}

func mockMultiWriterStore() (*MultiWriterAppStore, error) {
	memDb, _ := db.LoadMemDB()
	iavlStore, err := NewIAVLStore(memDb, 0, 0, 0)
	if err != nil {
		return nil, err
	}
	memDb, _ = db.LoadMemDB()
	evmStore := NewEvmStore(memDb, 100)
	multiWriterStore, err := NewMultiWriterAppStore(iavlStore, evmStore, false)
	if err != nil {
		return nil, err
	}
	return multiWriterStore, nil
}

func vmPrefixKey(key string) []byte {
	return util.PrefixKey([]byte("vm"), []byte(key))
}

func featureKey(key string) []byte {
	return util.PrefixKey(featurePrefix, []byte(key))
}

func configKey(key string) []byte {
	return util.PrefixKey(configPrefix, []byte(key))
}
