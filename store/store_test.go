package store

import (
	"bytes"
	"testing"
	"time"

	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

type storeTestFactory func(t *testing.T) (KVStore, string)

func TestStoreRange(t *testing.T) {
	factories := []storeTestFactory{
		func(t *testing.T) (KVStore, string) {
			db := dbm.NewMemDB()
			s, err := NewIAVLStore(db, 0)
			require.NoError(t, err)
			return s, "IAVLStore"
		},
		func(t *testing.T) (KVStore, string) {
			return NewMemStore(), "MemStore"
		},
	}

	for _, f := range factories {
		s, storeName := f(t)

		prefix1 := []byte("doremi")
		prefix2 := append([]byte("stop"), byte(255))
		prefix3 := append([]byte("stop"), byte(0))

		entries := []*plugin.RangeEntry{
			&plugin.RangeEntry{Key: util.PrefixKey([]byte("abc"), []byte("")), Value: []byte("1")},
			&plugin.RangeEntry{Key: util.PrefixKey([]byte("abc123"), []byte("")), Value: []byte("2")},

			&plugin.RangeEntry{Key: util.PrefixKey(prefix1, []byte("1")), Value: []byte("3")},
			&plugin.RangeEntry{Key: util.PrefixKey(prefix1, []byte("2")), Value: []byte("4")},
			&plugin.RangeEntry{Key: util.PrefixKey(prefix1, []byte("3")), Value: []byte("5")},

			&plugin.RangeEntry{Key: util.PrefixKey(prefix2, []byte("3")), Value: []byte("6")},
			&plugin.RangeEntry{Key: util.PrefixKey(prefix2, []byte("2")), Value: []byte("7")},
			&plugin.RangeEntry{Key: util.PrefixKey(prefix2, []byte("1")), Value: []byte("8")},
			&plugin.RangeEntry{Key: util.PrefixKey(prefix2, []byte("4")), Value: []byte("9")},

			&plugin.RangeEntry{Key: util.PrefixKey(prefix3, []byte{byte(0)}), Value: []byte("10")},
			&plugin.RangeEntry{Key: util.PrefixKey(prefix3, []byte{byte(255)}), Value: []byte("11")},
		}
		for _, e := range entries {
			s.Set(e.Key, e.Value)
		}
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
		require.Len(t, s.Range([]byte("abc123")), 1)
		require.EqualValues(t, []byte{}, s.Range([]byte("abc123"))[0].Key, storeName)
		require.EqualValues(t, entries[1].Value, s.Range([]byte("abc123"))[0].Value, storeName)

		key2, err := util.UnprefixKey(entries[2].Key, prefix1)
		require.NoError(t, err)
		key3, err := util.UnprefixKey(entries[3].Key, prefix1)
		require.NoError(t, err)
		key4, err := util.UnprefixKey(entries[4].Key, prefix1)
		require.NoError(t, err)

		expected := []*plugin.RangeEntry{
			{key2, entries[2].Value},
			{key3, entries[3].Value},
			{key4, entries[4].Value},
		}
		actual := s.Range(prefix1)
		require.Len(t, actual, len(expected), storeName)
		if storeName != "MemStore" {
			for i := range expected {
				require.EqualValues(t, expected[i], actual[i], storeName)
			}
		}

		key5, err := util.UnprefixKey(entries[5].Key, prefix2)
		require.NoError(t, err)
		key6, err := util.UnprefixKey(entries[6].Key, prefix2)
		require.NoError(t, err)
		key7, err := util.UnprefixKey(entries[7].Key, prefix2)
		require.NoError(t, err)
		key8, err := util.UnprefixKey(entries[8].Key, prefix2)
		require.NoError(t, err)
		expected = []*plugin.RangeEntry{
			{key7, entries[7].Value},
			{key6, entries[6].Value},
			{key5, entries[5].Value},
			{key8, entries[8].Value},
		}
		actual = s.Range(prefix2)
		require.Len(t, actual, len(expected), storeName)

		// TODO: MemStore keys should be iterated in ascending order
		if storeName != "MemStore" {
			for i := range expected {
				require.EqualValues(t, expected[i], actual[i], storeName)
			}
		}

		key9, err := util.UnprefixKey(entries[9].Key, prefix3)
		require.NoError(t, err)
		require.Equal(t, 0, bytes.Compare(key9, []byte{byte(0)}))
		key10, err := util.UnprefixKey(entries[10].Key, prefix3)
		require.NoError(t, err)
		require.Equal(t, 0, bytes.Compare(key10, []byte{byte(255)}))
		expected = []*plugin.RangeEntry{
			{key9, entries[9].Value},
			{key10, entries[10].Value},
		}
		actual = s.Range(prefix3)
		require.Len(t, actual, len(expected), storeName)
		if storeName != "MemStore" {
			for i := range expected {
				require.EqualValues(t, expected[i], actual[i], storeName)
			}
		}
	}
}

func TestPruningIAVLStoreBatching(t *testing.T) {
	db := dbm.NewMemDB()
	cfg := PruningIAVLStoreConfig{
		MaxVersions: 5,
		BatchSize:   5,
		Interval:    1 * time.Second,
	}
	store, err := NewPruningIAVLStore(db, cfg)
	require.NoError(t, err)

	require.Equal(t, int64(0), store.oldestVer)

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

	curVer := int64(1)
	for _, kv := range values {
		store.Set(kv.key, kv.val)
		_, ver, err := store.SaveVersion()
		require.NoError(t, err)
		require.Equal(t, curVer, ver)
		curVer++
	}

	time.Sleep(5 * time.Second)

	require.True(t, store.Version() > cfg.MaxVersions)
	require.Equal(t, store.Version(), store.oldestVer+cfg.MaxVersions-1, "correct number of versions has been kept")
	require.Equal(t, uint64(2), store.batchCount, "correct number of batches has been pruned")

	prevOldestVer := store.oldestVer

	store, err = NewPruningIAVLStore(db, cfg)
	require.NoError(t, err)

	// the oldest version shouldn't change when the IAVL store is reloaded
	require.Equal(t, prevOldestVer, store.oldestVer)
}

func TestPruningIAVLStoreKeepsAtLeastTwoVersions(t *testing.T) {
	cfg := PruningIAVLStoreConfig{
		MaxVersions: 1,
		BatchSize:   5,
		Interval:    1 * time.Second,
	}
	store, err := NewPruningIAVLStore(dbm.NewMemDB(), cfg)
	require.NoError(t, err)
	require.Equal(t, int64(0), store.Version())

	values := []struct {
		key []byte
		val []byte
	}{
		{key: key1, val: val1},
		{key: key2, val: val2},
	}

	for i, kv := range values {
		if i == 2 {
			break
		}

		store.Set(kv.key, kv.val)
		_, _, err := store.SaveVersion()
		require.NoError(t, err)
	}

	time.Sleep(5 * time.Second)

	require.Equal(t, int64(2), store.Version())
	require.Equal(t, int64(1), store.oldestVer)
	require.Equal(t, uint64(0), store.batchCount)
}

func TestPruningIAVLStoreKeepsAllVersionsIfMaxVersionsIsZero(t *testing.T) {
	cfg := PruningIAVLStoreConfig{
		MaxVersions: 0,
		BatchSize:   5,
		Interval:    1 * time.Second,
	}
	store, err := NewPruningIAVLStore(dbm.NewMemDB(), cfg)
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

	time.Sleep(4 * time.Second)

	require.Equal(t, int64(12), store.Version())
	require.Equal(t, uint64(0), store.batchCount)
}

func TestIAVLStoreKeepsAllVersionsIfMaxVersionsIsZero(t *testing.T) {
	store, err := NewIAVLStore(dbm.NewMemDB(), 0)
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

func TestSwitchFromIAVLStoreToPruningIAVLStore(t *testing.T) {
	memDB := dbm.NewMemDB()
	store1, err := NewIAVLStore(memDB, 0)
	require.NoError(t, err)

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
		store1.Set(kv.key, kv.val)
		_, _, err := store1.SaveVersion()
		require.NoError(t, err)
	}

	require.Equal(t, int64(12), store1.Version())

	store2, err := NewIAVLStore(memDB, 11)
	require.NoError(t, err)
	// force the store to prune an old version
	store2.Set(key1, val1)
	_, _, err = store2.SaveVersion()
	require.NoError(t, err)

	require.Equal(t, int64(13), store2.Version())

	cfg := PruningIAVLStoreConfig{
		MaxVersions: 5,
		BatchSize:   5,
		Interval:    1 * time.Second,
	}
	store3, err := NewPruningIAVLStore(memDB, cfg)
	require.NoError(t, err)

	time.Sleep(4 * time.Second)

	require.Equal(t, (store3.Version()-cfg.MaxVersions)+1, store3.oldestVer)
}
