package loomchain

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/loomnetwork/go-loom"

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
	pubKey, err := base64.StdEncoding.DecodeString("e8NSTtvP5KViOdeQWhSEglxOLuq8JW31IsmhqyGwUvQ=")
	addr := loom.Address{ChainID: "default", Local: loom.LocalAddressFromPublicKey(pubKey)}
	fmt.Println(addr.String())
	kvStore, err := mockMultiWriterStore(10)
	require.NoError(t, err)
	// This is the first version of on chain-config
	originalConfig := &cctypes.Config{
		AppStore: &cctypes.AppStoreConfig{
			NumEvmKeysToPrune: 777,
			IAVLFlushInterval: 50,
		},
	}
	require.NoError(t, store.SaveOnChainConfig(kvStore, originalConfig))

	curCfg, err := store.LoadOnChainConfig(kvStore)
	require.NoError(t, err)

	header := abci.Header{
		Height: blockHeight,
		Time:   blockTime,
	}
	state := NewStoreState(
		context.Background(), kvStore, header, nil, nil,
	).WithOnChainConfig(curCfg)
	// check default config
	require.Equal(t, uint64(777), state.Config().AppStore.NumEvmKeysToPrune)
	require.Equal(t, uint64(50), state.Config().AppStore.IAVLFlushInterval)
	require.NotNil(t, state.Config().GetEvm())
	require.Equal(t, uint64(0), state.Config().GetEvm().GasLimit)
	// change config setting
	err = state.ChangeConfigSetting("Evm.GasLimit", "5000")
	require.NoError(t, err)
	// reload config
	curCfg, err = store.LoadOnChainConfig(kvStore)
	require.NoError(t, err)
	require.Equal(t, uint64(5000), state.WithOnChainConfig(curCfg).Config().Evm.GasLimit)
}

func mockMultiWriterStore(flushInterval int64) (*store.MultiWriterAppStore, error) {
	memDb, _ := db.LoadMemDB()
	iavlStore, err := store.NewIAVLStore(memDb, 0, 0, flushInterval)
	if err != nil {
		return nil, err
	}
	memDb, _ = db.LoadMemDB()
	evmStore := store.NewEvmStore(memDb, 100)
	multiWriterStore, err := store.NewMultiWriterAppStore(iavlStore, evmStore, false)
	if err != nil {
		return nil, err
	}
	return multiWriterStore, nil
}
