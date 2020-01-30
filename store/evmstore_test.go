package store

import (
	"bytes"
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
	require.Equal(true, bytes.Equal(root, []byte{100}))
	require.Equal(int64(100), version)
}
