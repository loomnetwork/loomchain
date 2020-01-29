package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachingStoreVersion(t *testing.T) {
	defaultConfig := DefaultCachingStoreConfig()
	defaultConfig.CachingEnabled = true

	mockStore, err := mockMultiWriterStore(0, 0)
	require.NoError(t, err)

	key1 := []byte("key1")
	key2 := []byte("key2")
	key3 := []byte("key3")

	versionedStore, err := NewVersionedCachingStore(mockStore, defaultConfig, mockStore.Version())
	require.NoError(t, err)

	versionedStore.Set(key1, []byte("value1"))
	versionedStore.Set(key2, []byte("value2"))
	versionedStore.Set(key3, []byte("value3"))

	snapshotv0, err := versionedStore.GetSnapshotAt(0)
	require.NoError(t, err)

	// snapshot should be empty because values haven't been persisted to the underlying store
	cachedValue := snapshotv0.Get(key1)
	assert.Equal(t, "", string(cachedValue), "snapshot should be empty")

	_, version, _ := versionedStore.SaveVersion(nil)
	assert.Equal(t, int64(1), version, "version must be updated to 1")

	// previously obtained snapshot should still be empty since it shouldn't be affected by changes
	// to the underlying store
	cachedValue = snapshotv0.Get(key1)
	assert.Equal(t, "", string(cachedValue), "snapshot should be empty")

	snapshotv1, err := versionedStore.GetSnapshotAt(0)
	require.NoError(t, err)

	// new snapshot should contain the previously persisted values
	cachedValue = snapshotv1.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "value should match the one persisted to the underlying store")
	// existing snapshot should be unaffected by unpersisted changes to the store
	versionedStore.Set(key1, []byte("newvalue1"))
	cachedValue = snapshotv1.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "snapshot should not be affected by changes to the underlying store")

	// save to bump up version
	_, version, _ = versionedStore.SaveVersion(nil)
	assert.Equal(t, int64(2), version, "version must be updated to 2")

	// existing snapshot should be unaffected by persisted changes to the store
	cachedValue = snapshotv1.Get(key1)
	assert.Equal(t, "value1", string(cachedValue), "snapshot should not be affected by changes to the uderlying store")

	// save data into version 3
	versionedStore.Set(key2, []byte("newvalue2"))
	versionedStore.Set(key3, []byte("newvalue3"))

	_, version, _ = versionedStore.SaveVersion(nil)
	assert.Equal(t, int64(3), version, "version must be updated to 3")

	snapshotv2, err := versionedStore.GetSnapshotAt(0)
	require.NoError(t, err)
	cachedValue = snapshotv2.Get(key1)
	assert.Equal(t, "newvalue1", string(cachedValue))
	cachedValue = snapshotv2.Get(key2)
	assert.Equal(t, "newvalue2", string(cachedValue))
	cachedValue = snapshotv2.Get(key3)
	assert.Equal(t, "newvalue3", string(cachedValue))

	// snapshotv1 should remain unchanged
	cachedValue = snapshotv1.Get(key1)
	assert.Equal(t, "value1", string(cachedValue))
	cachedValue = snapshotv1.Get(key2)
	assert.Equal(t, "value2", string(cachedValue))
	cachedValue = snapshotv1.Get(key3)
	assert.Equal(t, "value3", string(cachedValue))
}
