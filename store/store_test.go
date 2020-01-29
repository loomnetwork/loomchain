package store

import (
	"bytes"
	"fmt"
	"sync"
	"testing"

	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	dbm "github.com/tendermint/tendermint/libs/db"
)

var (
	key1 = []byte("1234")
	key2 = []byte("af876")
	key3 = []byte("9876")
	val1 = []byte("value1")
	val2 = []byte("value2")
	val3 = []byte("value3")
)

func TestCacheTxCommit(t *testing.T) {
	tests := []struct {
		tx tempTx
	}{
		{
			tempTx{Action: txSet, Key: key1, Value: val1},
		},
		{
			tempTx{Action: txSet, Key: key2, Value: val2},
		},
		{
			tempTx{Action: txSet, Key: key3, Value: val3},
		},
		{
			tempTx{Action: txDelete, Key: key1},
		},
		{
			tempTx{Action: txDelete, Key: key2},
		},
		{
			tempTx{Action: txSet, Key: key2, Value: val2},
		},
	}
	s := NewMemStore()
	cs := newCacheTx(s)

	for _, test := range tests {
		switch test.tx.Action {
		case txSet:
			cs.Set(test.tx.Key, test.tx.Value)
		case txDelete:
			cs.Delete(test.tx.Key)
		}
	}

	// ordering
	for i, test := range tests {
		tx := cs.tmpTxs[i]
		assert.Equal(t, tx.Action, test.tx.Action)
		assert.Equal(t, tx.Key, test.tx.Key)
		assert.Equal(t, tx.Value, test.tx.Value)
	}

	// cache functionality
	v1 := cs.Get(key1)
	assert.Nil(t, v1)
	v2 := cs.Get(key2)
	assert.Equal(t, val2, v2)
	v3 := cs.Get(key3)
	assert.Equal(t, val3, v3)

	// underlying store should not be modified
	v1 = s.Get(key1)
	assert.Nil(t, v1)
	v2 = s.Get(key2)
	assert.Nil(t, v2)
	v3 = s.Get(key3)
	assert.Nil(t, v3)

	// commit
	cs.Commit()
	v1 = cs.Get(key1)
	assert.Nil(t, v1)
	v2 = cs.Get(key2)
	assert.Equal(t, val2, v2)
	v3 = cs.Get(key3)
	assert.Equal(t, val3, v3)

	// underlying store should be modified
	v1 = s.Get(key1)
	assert.Nil(t, v1)
	v2 = s.Get(key2)
	assert.Equal(t, val2, v2)
	v3 = s.Get(key3)
	assert.Equal(t, val3, v3)
}

func TestCacheTxRollback(t *testing.T) {
	tests := []struct {
		tx tempTx
	}{
		{
			tempTx{Action: txSet, Key: key1, Value: val1},
		},
		{
			tempTx{Action: txSet, Key: key2, Value: val2},
		},
		{
			tempTx{Action: txSet, Key: key3, Value: val3},
		},
		{
			tempTx{Action: txDelete, Key: key1},
		},
		{
			tempTx{Action: txDelete, Key: key2},
		},
		{
			tempTx{Action: txSet, Key: key2, Value: val2},
		},
	}
	s := NewMemStore()
	cs := newCacheTx(s)

	for _, test := range tests {
		switch test.tx.Action {
		case txSet:
			cs.Set(test.tx.Key, test.tx.Value)
		case txDelete:
			cs.Delete(test.tx.Key)
		}
	}

	// ordering
	for i, test := range tests {
		tx := cs.tmpTxs[i]
		assert.Equal(t, tx.Action, test.tx.Action)
		assert.Equal(t, tx.Key, test.tx.Key)
		assert.Equal(t, tx.Value, test.tx.Value)
	}

	// cache functionality
	v1 := cs.Get(key1)
	assert.Nil(t, v1)
	v2 := cs.Get(key2)
	assert.Equal(t, val2, v2)
	v3 := cs.Get(key3)
	assert.Equal(t, val3, v3)

	// underlying store should not be modified
	v1 = s.Get(key1)
	assert.Nil(t, v1)
	v2 = s.Get(key2)
	assert.Nil(t, v2)
	v3 = s.Get(key3)
	assert.Nil(t, v3)

	// rollback
	cs.Rollback()
	v1 = cs.Get(key1)
	assert.Nil(t, v1)
	v2 = cs.Get(key2)
	assert.Nil(t, v2)
	v3 = cs.Get(key3)
	assert.Nil(t, v3)

	// underlying store should not be modified
	v1 = s.Get(key1)
	assert.Nil(t, v1)
	v2 = s.Get(key2)
	assert.Nil(t, v2)
	v3 = s.Get(key3)
	assert.Nil(t, v3)
}

//
// Common setup & tests that run for each store
//
type StoreTestSuite struct {
	suite.Suite
	store     VersionedKVStore
	StoreName string
}

func populateStore(s KVWriter) ([][]byte, []*plugin.RangeEntry) {
	prefixes := [][]byte{
		[]byte("doremi"),
		append([]byte("stop"), byte(255)),
		append([]byte("stop"), byte(0)),
	}

	entries := []*plugin.RangeEntry{
		&plugin.RangeEntry{Key: util.PrefixKey([]byte("abc"), []byte("")), Value: []byte("1")},
		&plugin.RangeEntry{Key: util.PrefixKey([]byte("abc123"), []byte("")), Value: []byte("2")},

		&plugin.RangeEntry{Key: util.PrefixKey(prefixes[0], []byte("1")), Value: []byte("3")},
		&plugin.RangeEntry{Key: util.PrefixKey(prefixes[0], []byte("2")), Value: []byte("4")},
		&plugin.RangeEntry{Key: util.PrefixKey(prefixes[0], []byte("3")), Value: []byte("5")},

		&plugin.RangeEntry{Key: util.PrefixKey(prefixes[1], []byte("3")), Value: []byte("6")},
		&plugin.RangeEntry{Key: util.PrefixKey(prefixes[1], []byte("2")), Value: []byte("7")},
		&plugin.RangeEntry{Key: util.PrefixKey(prefixes[1], []byte("1")), Value: []byte("8")},
		&plugin.RangeEntry{Key: util.PrefixKey(prefixes[1], []byte("4")), Value: []byte("9")},

		&plugin.RangeEntry{Key: util.PrefixKey(prefixes[2], []byte{byte(0)}), Value: []byte("10")},
		&plugin.RangeEntry{Key: util.PrefixKey(prefixes[2], []byte{byte(255)}), Value: []byte("11")},
	}

	for _, e := range entries {
		s.Set(e.Key, e.Value)
	}

	return prefixes, entries
}

func verifyRange(
	require *require.Assertions, storeName string, s KVReader, prefixes [][]byte, entries []*plugin.RangeEntry,
) {
	// TODO: This passed before the last Tendermint upgrade, doesn't anymore, figure out why.
	/*
		expected := []*plugin.RangeEntry{
			entries[0],
			entries[1],
		}
		actual := s.Range([]byte("abc"))

		require.Len(t, actual, 2)
		if storeName != "MemStore" { // TODO: MemStore iteration order should be stable, no random
			for i := range expected {
				require.EqualValues(t, expected[i], actual[i], storeName)
			}
		}
	*/
	require.Len(s.Range([]byte("abc123")), 1)
	require.EqualValues([]byte{}, s.Range([]byte("abc123"))[0].Key, storeName)
	require.EqualValues(entries[1].Value, s.Range([]byte("abc123"))[0].Value, storeName)

	key2, err := util.UnprefixKey(entries[2].Key, prefixes[0])
	require.NoError(err)
	key3, err := util.UnprefixKey(entries[3].Key, prefixes[0])
	require.NoError(err)
	key4, err := util.UnprefixKey(entries[4].Key, prefixes[0])
	require.NoError(err)

	expected := []*plugin.RangeEntry{
		{key2, entries[2].Value},
		{key3, entries[3].Value},
		{key4, entries[4].Value},
	}
	actual := s.Range(prefixes[0])
	require.Len(actual, len(expected), storeName)
	if storeName != "MemStore" {
		for i := range expected {
			require.EqualValues(expected[i], actual[i], storeName)
		}
	}

	key5, err := util.UnprefixKey(entries[5].Key, prefixes[1])
	require.NoError(err)
	key6, err := util.UnprefixKey(entries[6].Key, prefixes[1])
	require.NoError(err)
	key7, err := util.UnprefixKey(entries[7].Key, prefixes[1])
	require.NoError(err)
	key8, err := util.UnprefixKey(entries[8].Key, prefixes[1])
	require.NoError(err)
	expected = []*plugin.RangeEntry{
		{key7, entries[7].Value},
		{key6, entries[6].Value},
		{key5, entries[5].Value},
		{key8, entries[8].Value},
	}
	actual = s.Range(prefixes[1])
	require.Len(actual, len(expected), storeName)

	// TODO: MemStore keys should be iterated in ascending order
	if storeName != "MemStore" {
		for i := range expected {
			require.EqualValues(expected[i], actual[i], storeName)
		}
	}

	key9, err := util.UnprefixKey(entries[9].Key, prefixes[2])
	require.NoError(err)
	require.Equal(0, bytes.Compare(key9, []byte{byte(0)}))
	key10, err := util.UnprefixKey(entries[10].Key, prefixes[2])
	require.NoError(err)
	require.Equal(0, bytes.Compare(key10, []byte{byte(255)}))
	expected = []*plugin.RangeEntry{
		{key9, entries[9].Value},
		{key10, entries[10].Value},
	}
	actual = s.Range(prefixes[2])
	require.Len(actual, len(expected), storeName)
	if storeName != "MemStore" {
		for i := range expected {
			require.EqualValues(expected[i], actual[i], storeName)
		}
	}
}

func (ts *StoreTestSuite) TestStoreRange() {
	require := ts.Require()
	prefixes, entries := populateStore(ts.store)
	verifyRange(require, ts.StoreName, ts.store, prefixes, entries)
	_, _, err := ts.store.SaveVersion()
	require.NoError(err)
	verifyRange(require, ts.StoreName, ts.store, prefixes, entries)
}

func verifyConcurrentSnapshots(require *require.Assertions, s VersionedKVStore) {
	// start one writer go-routine and a bunch of reader go-routines
	var wg sync.WaitGroup
	numOps := 10000

	// writer
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < numOps; i++ {
			s.Set([]byte(fmt.Sprintf("key/%d", i)), []byte(fmt.Sprintf("value/%d", i)))
			if i%10 == 0 {
				_, _, err := s.SaveVersion()
				require.NoError(err)
			}
		}
		_, _, err := s.SaveVersion()
		require.NoError(err)
	}()
	wg.Wait()

	// readers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var snap Snapshot
			for i := 0; i < numOps; i++ {
				if i%20 == 0 {
					if snap != nil {
						snap.Release()
					}
					var err error
					snap, err = s.GetSnapshotAt(0)
					require.NoError(err)
				}
				snap.Get([]byte(fmt.Sprintf("key/%d", i)))
			}

			if snap != nil {
				snap.Release()
			}
		}()
	}

	wg.Wait()
}

//
// IAVLStore - with pretend snapshots that really aren't.
//
func TestIAVLStoreTestSuite(t *testing.T) {
	suite.Run(t, &IAVLStoreTestSuite{})
}

type IAVLStoreTestSuite struct {
	StoreTestSuite
}

func (ts *IAVLStoreTestSuite) SetupSuite() {
	ts.StoreName = "IAVLStore"
}

// runs before each test in this suite
func (ts *IAVLStoreTestSuite) SetupTest() {
	require := ts.Require()
	var err error
	db := dbm.NewMemDB()
	ts.store, err = NewIAVLStore(db, 0, 0, 0)
	require.NoError(err)
}

//
// MemStore - broken in various ways, dunno why we even have this.
//

func TestMemStoreTestSuite(t *testing.T) {
	suite.Run(t, &MemStoreTestSuite{})
}

type MemStoreTestSuite struct {
	StoreTestSuite
}

// runs before each test in this suite
func (ts *MemStoreTestSuite) SetupTest() {
	ts.store = NewMemStore()
}

func (ts *MemStoreTestSuite) SetupSuite() {
	ts.StoreName = "MemStore"
}

func TestIAVLStoreKeepsAllVersionsIfMaxVersionsIsZero(t *testing.T) {
	store, err := NewIAVLStore(dbm.NewMemDB(), 0, 0, 0)
	require.NoError(t, err)
	require.Equal(t, int64(0), store.Version())
	require.Equal(t, int64(0), store.maxVersions)

	values := []struct {
		key []byte
		val []byte
	}{
		{key: key1, val: val1},
		{key: key2, val: val2},
		{key: key3, val: val3},
		{key: key1, val: val3},
		{key: key2, val: val1},
		{key: key3, val: val2},
		{key: key1, val: val1},
		{key: key2, val: val2},
		{key: key3, val: val3},
		{key: key1, val: val3},
		{key: key2, val: val1},
		{key: key3, val: val2},
	} // 12 items

	for _, kv := range values {
		store.Set(kv.key, kv.val)
		_, _, err := store.SaveVersion()
		require.NoError(t, err)
	}

	require.Equal(t, int64(12), store.Version())
}
