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
			IavlFlushInterval: 50,
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
	).WithOnChainConfig(loadOnChainConfig(kvStore))
	// check default config
	require.Equal(t, uint64(777), state.Config().AppStore.NumEvmKeysToPrune)
	require.Equal(t, uint64(50), state.Config().AppStore.IavlFlushInterval)
	require.NotNil(t, state.Config().GetEvm())
	require.Equal(t, uint64(0), state.Config().GetEvm().GasLimit)
	// change config setting
	err = state.ChangeConfigSetting("Evm.GasLimit", "5000")
	require.NoError(t, err)
	// realod config
	state = state.WithOnChainConfig(loadOnChainConfig(kvStore))
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
