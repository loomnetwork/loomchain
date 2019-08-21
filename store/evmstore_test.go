package store

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/loomnetwork/go-loom/util"
	"github.com/loomnetwork/loomchain/db"
	"github.com/stretchr/testify/suite"
)

type EvmStoreTestSuite struct {
	suite.Suite
}

func (t *EvmStoreTestSuite) SetupTest() {
}

func TestEvmStoreTestSuite(t *testing.T) {
	suite.Run(t, new(EvmStoreTestSuite))
}
func (t *EvmStoreTestSuite) TestEvmStoreRangeAndCommit() {
	require := t.Require()
	evmDb, err := db.LoadMemDB()
	require.NoError(err)
	evmStore := NewEvmStore(evmDb, 100)
	for i := 0; i <= 100; i++ {
		key := []byte(fmt.Sprintf("Key%d", i))
		evmStore.Set(key, key)
	}
	evmStore.Set([]byte("hellovm"), []byte("world"))
	evmStore.Set([]byte("hellovm"), []byte("world3"))
	evmStore.Set([]byte("hello1"), []byte("world1"))
	evmStore.Set([]byte("hello2"), []byte("world2"))
	evmStore.Set([]byte("hello3"), []byte("world3"))
	evmStore.Set([]byte("hello3"), []byte("world3"))
	evmStore.Delete([]byte("hello2"))
	dataRange := evmStore.Range(nil)
	require.Equal(104, len(dataRange))
	require.Equal(false, evmStore.Has([]byte("hello2")))
	evmStore.Commit(1)
	evmStore.Set([]byte("SSSSS"), []byte("SSSSS"))
	evmStore.Set([]byte("vvvvv"), []byte("vvv"))
	dataRange = evmStore.Range(nil)
	require.Equal(107, len(dataRange))
	evmStore.Commit(2)
	evmStore.Set([]byte("SSSSS"), []byte("S1"))
	ret := evmStore.Get([]byte("SSSSS"))
	require.Equal(0, bytes.Compare(ret, []byte("S1")))
	evmStore.Delete([]byte("SSSSS"))
	evmStore.Delete([]byte("hello1"))
	dataRange = evmStore.Range(nil)
	require.Equal(105, len(dataRange))
	evmStore.Commit(3)
	evmStore.Delete([]byte("SSSSS"))
	evmStore.Delete([]byte("hello1"))
	dataRange = evmStore.Range(nil)
	require.Equal(105, len(dataRange))
}

func (t *EvmStoreTestSuite) TestEvmStoreBasicMethods() {
	require := t.Require()
	// Test Get|Set|Has|Delete methods
	evmDb, err := db.LoadMemDB()
	require.NoError(err)
	evmStore := NewEvmStore(evmDb, 100)
	key1 := []byte("hello")
	key2 := []byte("hello2")
	value1 := []byte("world")
	value2 := []byte("world2")
	value3 := []byte("This is a new value")
	evmStore.Set(key1, value1)
	evmStore.Set(key2, value2)
	result := evmStore.Get(key1)
	require.Equal(0, bytes.Compare(value1, result))
	evmStore.Set(key1, value3)
	result = evmStore.Get(key1)
	require.Equal(0, bytes.Compare(value3, result))
	has := evmStore.Has(key1)
	require.Equal(true, has)
	evmStore.Delete(key1)
	has = evmStore.Has(key1)
	require.Equal(false, has)
	result = evmStore.Get(key1)
	fmt.Println(result)
	require.Equal(0, len(result))
}

func (t *EvmStoreTestSuite) TestEvmStoreRangePrefix() {
	require := t.Require()
	// Test Range Prefix
	evmDb, err := db.LoadMemDB()
	require.NoError(err)
	evmStore := NewEvmStore(evmDb, 100)
	for i := 0; i <= 100; i++ {
		key := []byte(fmt.Sprintf("Key%d", i))
		evmStore.Set(key, key)
	}
	for i := 0; i <= 100; i++ {
		key := []byte(fmt.Sprintf("vv%dKey", i))
		evmStore.Set(key, key)
	}
	dataRange := evmStore.Range(nil)
	require.Equal(202, len(dataRange))

	dataRange = evmStore.Range([]byte("Key"))
	require.Equal(0, len(dataRange))

	for i := 0; i <= 100; i++ {
		key := util.PrefixKey([]byte("Key"), []byte(fmt.Sprintf("%d", i)))
		evmStore.Set(key, key)
		key = util.PrefixKey([]byte("vv"), []byte(fmt.Sprintf("%d", i)))
		evmStore.Set(key, key)
	}

	dataRange = evmStore.Range([]byte("Key"))
	require.Equal(101, len(dataRange))

	dataRange = evmStore.Range([]byte("vv"))
	require.Equal(101, len(dataRange))

	evmStore.Commit(1)
	dataRange = evmStore.Range([]byte("Key"))
	require.Equal(101, len(dataRange))

	dataRange = evmStore.Range([]byte("vv"))
	require.Equal(101, len(dataRange))

	evmStore.Commit(2)
	evmStore.Delete(util.PrefixKey([]byte("vv"), []byte(fmt.Sprintf("%d", 10))))
	dataRange = evmStore.Range([]byte("vv"))
	require.Equal(100, len(dataRange))
}

func (t *EvmStoreTestSuite) TestLoadVersionEvmStore() {
	require := t.Require()
	evmDb, err := db.LoadMemDB()
	require.NoError(err)
	evmDb.Set(evmRootKey(1), []byte{1})
	evmDb.Set(evmRootKey(2), []byte{2})
	evmDb.Set(evmRootKey(3), []byte{3})
	evmDb.Set(evmRootKey(100), []byte{100})
	evmDb.Set(evmRootKey(200), []byte{200})

	evmStore := NewEvmStore(evmDb, 100)
	err = evmStore.LoadVersion(500)
	require.NoError(err)
	root, version := evmStore.Version()
	require.Equal(true, bytes.Equal(root, []byte{200}))
	require.Equal(int64(200), version)

	require.NoError(evmStore.LoadVersion(2))
	root, version = evmStore.Version()
	require.Equal(true, bytes.Equal(root, []byte{2}))
	require.Equal(int64(2), version)

	require.NoError(evmStore.LoadVersion(99))
	root, version = evmStore.Version()
	require.Equal(true, bytes.Equal(root, []byte{3}))
	require.Equal(int64(3), version)

	require.NoError(evmStore.LoadVersion(100))
	root, version = evmStore.Version()
	require.Equal(true, bytes.Equal(root, []byte{100}))
	require.Equal(int64(100), version)
}
