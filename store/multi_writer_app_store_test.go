package store

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/loomnetwork/go-loom/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/tendermint/iavl"

	"github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/log"
)

var (
	temp []byte
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
	store, err := mockMultiWriterStore(10)
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
	store, err := mockMultiWriterStore(10)
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
	_, version, err = store.SaveVersion()
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
	store, err := mockMultiWriterStore(10)
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

func mockMultiWriterStore(flushInterval int64) (*MultiWriterAppStore, error) {
	memDb, _ := db.LoadMemDB()
	iavlStore, err := NewIAVLStore(memDb, 0, 0, flushInterval)
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

func TestMultiWriterSnapshots(t *testing.T) {
	numBlocks = 25
	blockSize = 5
	flushInterval := int64(10)
	prefix := []byte("test")
	log.Setup("debug", "file://-")
	log.Root.With("module", "dual-iavlstore")

	blocks = nil
	blocks = iavl.GenerateBlocksHashKeys(numBlocks, blockSize, append(prefix, byte(0)))

	controlStore, err := mockMultiWriterStore(flushInterval)
	var controlHashes [][]byte
	require.NoError(t, err)
	for _, block := range blocks {
		require.NoError(t, block.Execute(controlStore.appStore.tree))
		hash, _, err := controlStore.SaveVersion()
		controlHashes = append(controlHashes, hash)
		require.NoError(t, err)
	}

	testStore, err := mockMultiWriterStore(flushInterval)
	require.NoError(t, err)

	quit := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(2)
	var testHashes [][]byte

	go func(s *MultiWriterAppStore) {
		defer wg.Done()
		for _, block := range blocks {
			require.NoError(t, block.Execute(s.appStore.tree))
			hash, _, err := s.SaveVersion()
			require.NoError(t, err)
			testHashes = append(testHashes, hash)
			time.Sleep(5 * time.Millisecond)
		}
		quit <- true
	}(testStore)

	go func(s *MultiWriterAppStore) {
		defer wg.Done()
		for {
			select {
			case <-quit:
				return
			default:
				{
					snapshot := s.GetSnapshot()
					for _, data := range snapshot.Range(prefix) {
						temp = data.Key
						temp = data.Value
					}
					time.Sleep(time.Millisecond)
					snapshot.Release()
				}
			}
		}

	}(testStore)

	wg.Wait()

	for _, data := range testStore.Range(prefix) {
		value := controlStore.Get(util.PrefixKey(prefix, data.Key))
		require.Zero(t, bytes.Compare(value, data.Value))
	}
	for _, data := range controlStore.Range(prefix) {
		value := testStore.Get(util.PrefixKey(prefix, data.Key))
		require.Zero(t, bytes.Compare(value, data.Value))
	}
	require.Equal(t, len(controlHashes), numBlocks)
	require.Equal(t, len(controlHashes), len(testHashes))
	for i := range controlHashes {
		require.Zero(t, bytes.Compare(controlHashes[i], testHashes[i]))
	}
}
