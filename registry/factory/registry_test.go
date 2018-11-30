package factory

import (
	"context"
	"testing"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"

	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	owner = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	contract1 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	contract2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
)

/*
type Registry interface {
	Register(contractName string, contractAddr, ownerAddr loom.Address) error
	Resolve(contractName string) (loom.Address, error)
	GetRecord(contractAddr loom.Address) (*Record, error)
	GetRecords(active bool) ([]*Record, error)
	SetActive(loom.Address) error
	SetInactive(loom.Address) error
	IsActive(loom.Address) bool
}
*/

func TestActiveInactive(t *testing.T) {
	createRegistry, err := NewRegistryFactory(RegistryV2)
	require.NoError(t, err)
	state := loomchain.NewStoreState(context.Background(), store.NewMemStore(), abci.Header{}, nil)
	reg := createRegistry(state)

	_, err = reg.GetRecord(contract1)
	require.Error(t, err)
	require.NoError(t, reg.Register("Contract1", contract1, owner))
	_, err = reg.GetRecord(contract1)
	require.NoError(t, err)

	require.NoError(t, reg.Register("Contract2", contract2, owner))

	records, err := reg.GetRecords(true)
	require.NoError(t, err)
	require.EqualValues(t, 2, len(records))

	records, err = reg.GetRecords(false)
	require.NoError(t, err)
	require.EqualValues(t, 0, len(records))

	require.True(t, reg.IsActive(contract1))
	require.NoError(t, reg.SetInactive(contract1))
	require.False(t, reg.IsActive(contract1))

	records, err = reg.GetRecords(false)
	require.NoError(t, err)
	require.Equal(t, 1, len(records))
	require.Equal(t, "Contract1" ,records[0].Name)

	records, err = reg.GetRecords(true)
	require.NoError(t, err)
	require.EqualValues(t, 1, len(records))
	require.Equal(t, "Contract2" ,records[0].Name)

	require.NoError(t, reg.SetActive(contract1))
	require.True(t, reg.IsActive(contract1))
	records, err = reg.GetRecords(false)
	require.NoError(t, err)
	require.Equal(t, 0, len(records))

	records, err = reg.GetRecords(true)
	require.NoError(t, err)
	require.EqualValues(t, 2, len(records))
}