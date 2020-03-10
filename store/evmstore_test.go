package store

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"

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

func (t *EvmStoreTestSuite) TestLoadVersionEvmStore() {
	require := t.Require()
	evmDb, err := db.LoadMemDB()
	require.NoError(err)
	evmDb.Set(evmRootKey(1), []byte{1})
	evmDb.Set(evmRootKey(2), []byte{2})
	evmDb.Set(evmRootKey(3), []byte{3})
	evmDb.Set(evmRootKey(100), []byte{100})
	evmDb.Set(evmRootKey(200), []byte{200})

	evmStore := NewEvmStore(evmDb, 100, 0)
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
	fmt.Println("version ", version)
	fmt.Println("root hash,", root)
	require.Equal(true, bytes.Equal(root, []byte{100}))
	require.Equal(int64(100), version)
}

func (t *EvmStoreTestSuite) TestEvmStoreFlushInterval() {
	require := t.Require()
	evmDB, err := db.LoadMemDB()
	require.NoError(err)

	var flushInterval int64 = 10
	var numRootCached int = 5
	evmStore := NewEvmStore(evmDB, numRootCached, flushInterval)

	err = evmStore.LoadVersion(0)
	require.NoError(err)
	rh, version := evmStore.Version()
	require.Equal(int64(0), version)
	require.Equal(0, len(rh))

	// Set current root and commit for 10 versions
	var flushVersion int64
	for i := int64(1); i <= flushInterval; i++ {
		s := strconv.FormatInt(i, 10)
		evmStore.SetCurrentRoot([]byte(s))
		evmStore.Commit(i, flushInterval)

		rh, version = evmStore.Version()
		require.Equal(i, version)
		require.Equal([]byte(s), rh)
		flushVersion = version
	}

	// Get 5 latest version, These version should come from rootCache
	for i := int64(6); i <= flushVersion; i++ {
		rh, version := evmStore.GetRootAt(i)
		require.Equal(i, version)
		require.True(len(rh) > 0)
	}

	// Set another 10 version to trigger 2nd flush.
	for i := int64(11); i <= int64(20); i++ {
		s := strconv.FormatInt(i, 10)
		evmStore.SetCurrentRoot([]byte(s))
		evmStore.Commit(i, flushInterval)
		rh, version = evmStore.Version()
		require.Equal(i, version)
		require.Equal([]byte(s), rh)
	}

	// Try to get 1st flushed version this should still be available.
	rh, version = evmStore.GetRootAt(10)
	require.Equal(int64(10), version)
	require.True(len(rh) > 0)

	// version 9 should gone by now.
	rh, version = evmStore.GetRootAt(9)
	require.Equal(int64(0), version)
	require.True(len(rh) == 0)
}
