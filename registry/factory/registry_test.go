package factory

import (
	"context"
	"testing"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	owner = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	contract1 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	contract2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
)

func TestActiveInactive(t *testing.T) {
	createRegistry, err := NewRegistryFactory(registry.RegistryV2)
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

	c1Addr, err := reg.Resolve("Contract1"); c1Addr = c1Addr
	require.NoError(t, err)
	require.Equal(t, 0, contract1.Compare(c1Addr))

	require.True(t, reg.IsActive(contract1))
	require.NoError(t, reg.SetInactive(contract1))
	require.False(t, reg.IsActive(contract1))

	records, err = reg.GetRecords(false)
	require.NoError(t, err)
	require.Equal(t, 1, len(records))
	require.Equal(t, "Contract1" ,records[0].Name)

	c1Addr, err = reg.Resolve("Contract1")
	require.NoError(t, err)

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

	c1Addr, err = reg.Resolve("Contract1");
	require.NoError(t, err)
	require.Equal(t, 0, contract1.Compare(c1Addr))
}