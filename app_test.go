package loomchain

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
	"github.com/loomnetwork/loomchain/db"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	blockHeight = int64(34)
	blockTime   = time.Unix(123456789, 0)
)

func TestOnChainConfig(t *testing.T) {
	kvStore, _, _, err := mockMultiWriterStore(10)
	require.NoError(t, err)
	// This is the first version of on chain-config
	originalConfig := &cctypes.Config{
		AppStore: &cctypes.AppStoreConfig{
			NumEvmKeysToPrune: 777,
		},
	}
	configBytes, err := proto.Marshal(originalConfig)
	kvStore.Set([]byte(configKey), configBytes)
	require.NoError(t, err)

	header := abci.Header{}
	header.Height = blockHeight
	header.Time = blockTime
	state := NewStoreState(
		context.Background(), kvStore, header, nil, nil,
	).WithOnChainConfig(loadOnChainConfig(kvStore, false))
	require.Equal(t, uint64(777), state.Config().AppStore.NumEvmKeysToPrune)
	// have not enabled chainconfig v1.4 so Evm is nil
	require.Nil(t, state.Config().Evm)

	// enable chainconfig v1.4
	state = state.WithOnChainConfig(loadOnChainConfig(kvStore, true))
	require.Equal(t, uint64(777), state.Config().AppStore.NumEvmKeysToPrune)
	require.NotNil(t, state.Config().Evm)
	require.Equal(t, uint64(0), state.Config().Evm.GasLimit)
	state.ChangeConfigSetting("Evm.GasLimit", "5000")
	require.Equal(t, uint64(5000), state.Config().Evm.GasLimit)
}

func mockMultiWriterStore(flushInterval int64) (*store.MultiWriterAppStore, *store.IAVLStore, *store.EvmStore, error) {
	memDb, _ := db.LoadMemDB()
	iavlStore, err := store.NewIAVLStore(memDb, 0, 0, flushInterval)
	if err != nil {
		return nil, nil, nil, err
	}
	memDb, _ = db.LoadMemDB()
	evmStore := store.NewEvmStore(memDb, 100)
	multiWriterStore, err := store.NewMultiWriterAppStore(iavlStore, evmStore, false)
	if err != nil {
		return nil, nil, nil, err
	}
	return multiWriterStore, iavlStore, evmStore, nil
}
