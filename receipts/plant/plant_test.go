package plant

import (
	`context`
	`github.com/loomnetwork/loomchain`
	`github.com/loomnetwork/loomchain/events`
	`github.com/loomnetwork/loomchain/registry/factory`
	`github.com/loomnetwork/loomchain/store`
	`github.com/stretchr/testify/require`
	`testing`
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestPlant(t *testing.T) {
	eventHandler := loomchain.NewDefaultEventHandler(events.NewLogEventDispatcher())
	createRegistry, err := factory.NewRegistryFactory(factory.LatestRegistryVersion)
	require.NoError(t,err)
	plant := NewReceiptPlant(eventHandler, createRegistry)
	plant = plant
}

func mockState(height int64) loomchain.State {
	header := abci.Header{}
	header.Height = height
	return loomchain.NewStoreState(context.Background(), store.NewMemStore(), header)
}