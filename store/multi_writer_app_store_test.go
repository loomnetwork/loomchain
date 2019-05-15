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
	if err != nil {
		panic(err)
	}
	store.evmStoreEnabled = true
	store.Set(vmPrefixKey("abcd"), []byte("hello"))
	store.Set(vmPrefixKey("abcde"), []byte("world"))
	store.Set(vmPrefixKey("evmStore"), []byte("yes"))
	store.Set(vmPrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("abcd"), []byte("NewData"))

	rangeData := store.Range(vmPrefix)
	require.Equal(4, len(rangeData))

	store.evmStoreEnabled = false
	rangeData = store.Range(vmPrefix)
	require.Equal(0, len(rangeData))

	store.Set(vmPrefixKey("gg"), []byte("world"))
	store.Set(vmPrefixKey("dd"), []byte("yes"))
	store.Set(vmPrefixKey("vv"), []byte("yes"))
	rangeData = store.Range(vmPrefix)
	require.Equal(3, len(rangeData))

	store.evmStoreEnabled = true
	rangeData = store.Range(vmPrefix)
	require.Equal(7, len(rangeData))

	rangeData = store.Range(nil)
	require.Equal(8, len(rangeData))

	store.evmStoreEnabled = false
	rangeData = store.Range(nil)
	require.Equal(4, len(rangeData))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreDelete() {
	require := m.Require()
	store, err := mockMultiWriterStore()
	if err != nil {
		panic(err)
	}
	store.evmStoreEnabled = true
	store.Set(vmPrefixKey("abcd"), []byte("hello"))
	store.Set(vmPrefixKey("abcde"), []byte("world"))
	store.Set(vmPrefixKey("evmStore"), []byte("yes"))
	store.Set(vmPrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("vmroot"), []byte("SSSSSSSSSSSSS"))
	store.Set([]byte("abcd"), []byte("NewData"))
	store.Delete(vmPrefixKey("abcd"))
	require.Equal(false, store.Has(vmPrefixKey("abcd")))

	rangeData := store.Range(vmPrefix)
	require.Equal(3, len(rangeData))

	store.evmStoreEnabled = false
	rangeData = store.Range(vmPrefix)
	require.Equal(0, len(rangeData))
	vmRoot := store.Get([]byte("vmroot"))
	require.Equal(0, bytes.Compare([]byte("SSSSSSSSSSSSS"), vmRoot))

	store.Set(vmPrefixKey("gg"), []byte("world"))
	store.Set(vmPrefixKey("dd"), []byte("yes"))
	store.Set(vmPrefixKey("vv"), []byte("yes"))
	store.Delete(vmPrefixKey("vv"))
	require.Equal(false, store.Has(vmPrefixKey("vv")))

	rangeData = store.Range(vmPrefix)
	require.Equal(2, len(rangeData))

	store.evmStoreEnabled = true
	rangeData = store.Range(vmPrefix)
	require.Equal(5, len(rangeData))

	store.evmStoreEnabled = false
	rangeData = store.Range(nil)
	require.Equal(4, len(rangeData))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSnapShot() {
	require := m.Require()
	store, err := mockMultiWriterStore()
	require.NoError(err)
	store.evmStoreEnabled = true
	store.Set(vmPrefixKey("abcd"), []byte("hello"))
	store.Set(vmPrefixKey("abcde"), []byte("world"))
	store.Set(vmPrefixKey("evmStore"), []byte("yes"))
	store.Set(vmPrefixKey("aaaa"), []byte("yes"))
	store.Set([]byte("ssssvvv"), []byte("SSSSSSSSSSSSS"))
	store.Set([]byte("abcd"), []byte("NewData"))
	_, saveVersion, _ := store.SaveVersion()

	snapshot := store.GetSnapshot()
	store.Set(vmPrefixKey("abcd"), []byte("hellooooooo"))
	store.Set(vmPrefixKey("abcde"), []byte("vvvvvvvvv"))
	store.Set([]byte("abcd"), []byte("asdfasdf"))
	snapshot = store.GetSnapshot()
	abcd := snapshot.Get([]byte("abcd"))
	abcdvm := snapshot.Get(vmPrefixKey("abcd"))
	abcde := snapshot.Get(vmPrefixKey("abcde"))
	require.Equal(0, bytes.Compare([]byte("hello"), abcdvm))
	require.Equal(0, bytes.Compare([]byte("NewData"), abcd))
	require.Equal(0, bytes.Compare([]byte("world"), abcde))

	store.SaveVersion()
	snapshot = store.GetSnapshot()
	abcd = snapshot.Get([]byte("abcd"))
	abcdvm = snapshot.Get(vmPrefixKey("abcd"))
	abcde = snapshot.Get(vmPrefixKey("abcde"))
	require.Equal(0, bytes.Compare([]byte("asdfasdf"), abcd))
	require.Equal(0, bytes.Compare([]byte("hellooooooo"), abcdvm))
	require.Equal(0, bytes.Compare([]byte("vvvvvvvvv"), abcde))

	// This one only has effect on app.db (IAVL tree)
	store.setLastSavedTreeToVersion(saveVersion)
	snapshot = store.GetSnapshot()
	abcd = snapshot.Get([]byte("abcd"))
	abcdvm = snapshot.Get(vmPrefixKey("abcd"))
	abcde = snapshot.Get(vmPrefixKey("abcde"))
	require.Equal(0, bytes.Compare([]byte("hellooooooo"), abcdvm))
	require.Equal(0, bytes.Compare([]byte("NewData"), abcd))
	require.Equal(0, bytes.Compare([]byte("vvvvvvvvv"), abcde))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSnapShotRange() {
	require := m.Require()
	store, err := mockMultiWriterStore()
	require.NoError(err)
	store.evmStoreEnabled = true
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
	store.SaveVersion()
	snapshot = store.GetSnapshot()
	rangeData = snapshot.Range(vmPrefix)
	require.Equal(5, len(rangeData))
	rangeData = snapshot.Range(nil)
	require.Equal(10, len(rangeData))

	store.Delete(vmPrefixKey("abcd"))
	store.Delete([]byte("ssssvvv"))
	snapshot = store.GetSnapshot()
	rangeData = snapshot.Range(vmPrefix)
	require.Equal(5, len(rangeData))
	rangeData = snapshot.Range(nil)
	require.Equal(10, len(rangeData))

	store.SaveVersion()
	snapshot = store.GetSnapshot()
	rangeData = snapshot.Range(vmPrefix)
	require.Equal(5, len(rangeData))
	rangeData = snapshot.Range(nil)
	require.Equal(9, len(rangeData))

	abcd := snapshot.Get(vmPrefixKey("abcd"))
	ssssvvv := snapshot.Get(vmPrefixKey("ssssvvv"))
	require.Equal(0, bytes.Compare([]byte(""), abcd))
	require.Equal(0, bytes.Compare([]byte(""), ssssvvv))
}

func (m *MultiWriterAppStoreTestSuite) TestMultiWriterAppStoreSaveVersion() {
	require := m.Require()
	store, err := mockMultiWriterStore()
	require.NoError(err)
	store.evmStoreEnabled = true
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
	result := store.Get(vmPrefixKey("abcd"))
	require.Equal(0, bytes.Compare([]byte("hello"), result))
	result = store.Get([]byte("abcd"))
	require.Equal(0, bytes.Compare([]byte("NewData"), result))
	require.Equal(true, store.Has(vmPrefixKey("gg")))
	store.Delete(vmPrefixKey("gg"))

	dataRange := store.Range(nil)
	require.Equal(10, len(dataRange))

	_, version, err = store.SaveVersion()
	require.Equal(int64(2), version)
	require.NoError(err)
	result = store.Get(vmPrefixKey("abcd"))
	require.Equal(0, bytes.Compare([]byte("hello"), result))
	result = store.Get([]byte("abcd"))
	require.Equal(0, bytes.Compare([]byte("NewData"), result))
	require.Equal(false, store.Has(vmPrefixKey("gg")))

	dataRange = store.Range(nil)
	require.Equal(11, len(dataRange))
}

func mockMultiWriterStore() (*MultiWriterAppStore, error) {
	memDb, _ := db.LoadMemDB()
	iavlStore, err := NewIAVLStore(memDb, 0, 0)
	if err != nil {
		return nil, err
	}
	memDb, _ = db.LoadMemDB()
	evmStore := NewEvmStore(memDb)
	multiWriterStore, err := NewMultiWriterAppStore(iavlStore, evmStore, false)
	if err != nil {
		return nil, err
	}
	return multiWriterStore, nil
}

func vmPrefixKey(key string) []byte {
	return util.PrefixKey([]byte("vm"), []byte(key))
}
