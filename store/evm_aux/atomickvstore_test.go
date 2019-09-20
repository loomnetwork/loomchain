package evmaux

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	dbm "github.com/tendermint/tendermint/libs/db"
)

func TestAtomicKVStore(t *testing.T) {
	memDB := dbm.NewMemDB()
	memDB.Set([]byte("Key1"), []byte("Value1"))
	memDB.Set([]byte("Key2"), []byte("Value2"))
	memDB.Set([]byte("Key3"), []byte("Value3"))
	atomicStore := NewAtomicKVStore(memDB)
	atomicStore.Set([]byte("Key4"), []byte("Value4"))
	atomicStore.Set([]byte("Key5"), []byte("Value5"))
	atomicStore.Set([]byte("Key6"), []byte("Value6"))

	// Check if all keys are store
	require.Equal(t, 0, bytes.Compare([]byte("Value1"), atomicStore.Get([]byte("Key1"))))
	require.Equal(t, 0, bytes.Compare([]byte("Value2"), atomicStore.Get([]byte("Key2"))))
	require.Equal(t, 0, bytes.Compare([]byte("Value3"), atomicStore.Get([]byte("Key3"))))
	require.Equal(t, 0, bytes.Compare([]byte("Value4"), atomicStore.Get([]byte("Key4"))))
	require.Equal(t, 0, bytes.Compare([]byte("Value5"), atomicStore.Get([]byte("Key5"))))
	require.Equal(t, 0, bytes.Compare([]byte("Value6"), atomicStore.Get([]byte("Key6"))))

	atomicStore.Delete([]byte("Key1"))
	atomicStore.Delete([]byte("Key4"))
	atomicStore.Set([]byte("Key2"), []byte("NewValue2"))
	atomicStore.Set([]byte("Key5"), []byte("NewValue5"))

	// Both keys are gone from atomicStore
	require.Equal(t, 0, bytes.Compare(nil, atomicStore.Get([]byte("Key1"))))
	require.Equal(t, 0, bytes.Compare(nil, atomicStore.Get([]byte("Key4"))))
	require.False(t, atomicStore.Has([]byte("Key1")))
	require.False(t, atomicStore.Has([]byte("Key4")))

	// Keys are updated
	require.Equal(t, 0, bytes.Compare([]byte("NewValue2"), atomicStore.Get([]byte("Key2"))))
	require.Equal(t, 0, bytes.Compare([]byte("NewValue5"), atomicStore.Get([]byte("Key5"))))

	// Key1 is still in DB
	require.Equal(t, 0, bytes.Compare([]byte("Value1"), memDB.Get([]byte("Key1"))))
	require.True(t, memDB.Has([]byte("Key1")))
	require.False(t, memDB.Has([]byte("Key4")))
	atomicStore.Commit()

	// Check all keys from atomicStore
	require.Equal(t, 0, bytes.Compare(nil, atomicStore.Get([]byte("Key1"))))
	require.False(t, atomicStore.Has([]byte("Key1")))
	require.Equal(t, 0, bytes.Compare([]byte("NewValue2"), atomicStore.Get([]byte("Key2"))))
	require.Equal(t, 0, bytes.Compare([]byte("Value3"), atomicStore.Get([]byte("Key3"))))
	require.Equal(t, 0, bytes.Compare(nil, atomicStore.Get([]byte("Key4"))))
	require.False(t, atomicStore.Has([]byte("Key4")))
	require.Equal(t, 0, bytes.Compare([]byte("NewValue5"), atomicStore.Get([]byte("Key5"))))
	require.Equal(t, 0, bytes.Compare([]byte("Value6"), atomicStore.Get([]byte("Key6"))))

	// Check all keys from memDB
	require.Equal(t, 0, bytes.Compare(nil, memDB.Get([]byte("Key1"))))
	require.False(t, memDB.Has([]byte("Key1")))
	require.Equal(t, 0, bytes.Compare([]byte("NewValue2"), memDB.Get([]byte("Key2"))))
	require.Equal(t, 0, bytes.Compare([]byte("Value3"), memDB.Get([]byte("Key3"))))
	require.Equal(t, 0, bytes.Compare(nil, memDB.Get([]byte("Key4"))))
	require.False(t, memDB.Has([]byte("Key4")))
	require.Equal(t, 0, bytes.Compare([]byte("NewValue5"), memDB.Get([]byte("Key5"))))
	require.Equal(t, 0, bytes.Compare([]byte("Value6"), memDB.Get([]byte("Key6"))))
}
