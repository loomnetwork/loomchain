package tx_handler

import (
	"context"
	"testing"

	proto "github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/feature"
	"github.com/loomnetwork/loomchain/migrations"
	"github.com/loomnetwork/loomchain/store"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	origin = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestMigrationTxHandler(t *testing.T) {
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil, nil)
	state.SetFeature(feature.MigrationTxFeature, true)

	ctx := context.WithValue(state.Context(), auth.ContextKeyOrigin, origin)
	s := state.WithContext(ctx)

	migrationTx1 := mockMessageTx(t, uint32(1), origin, origin)

	migrationFuncs := map[int32]MigrationFunc{
		1: func(ctx *migrations.MigrationContext) error { return nil },
		2: func(ctx *migrations.MigrationContext) error { return nil },
		3: func(ctx *migrations.MigrationContext) error { return nil },
	}

	migrationTxHandler := &MigrationTxHandler{
		Manager:        nil,
		CreateRegistry: nil,
		Migrations:     migrationFuncs,
	}

	state.SetFeature(feature.MigrationTxFeature, true)
	state.SetFeature(feature.MigrationFeaturePrefix+"1", true)
	_, err := migrationTxHandler.ProcessTx(s, migrationTx1, false)
	require.NoError(t, err)

	_, err = migrationTxHandler.ProcessTx(s, migrationTx1, false)
	require.Error(t, err)

	migrationTx2 := mockMessageTx(t, uint32(2), origin, origin)
	migrationTx4 := mockMessageTx(t, uint32(4), origin, origin)

	//Expect an error if migrationtx feature is not enabled
	state.SetFeature(feature.MigrationTxFeature, false)
	_, err = migrationTxHandler.ProcessTx(s, migrationTx2, false)
	require.Error(t, err)

	//Expect an error if migration id is not found
	state.SetFeature(feature.MigrationTxFeature, true)
	_, err = migrationTxHandler.ProcessTx(s, migrationTx4, false)
	require.Error(t, err)

	//Expect an error if migration feature is not enabled
	_, err = migrationTxHandler.ProcessTx(s, migrationTx2, false)
	require.Error(t, err)

	state.SetFeature(feature.MigrationTxFeature, true)
	state.SetFeature(feature.MigrationFeaturePrefix+"2", true)
	_, err = migrationTxHandler.ProcessTx(s, migrationTx2, false)
	require.NoError(t, err)

}

func mockMessageTx(t *testing.T, id uint32, to loom.Address, from loom.Address) []byte {
	var messageTx []byte

	migrationTx, err := proto.Marshal(&vm.MigrationTx{
		ID: id,
	})
	require.NoError(t, err)

	messageTx, err = proto.Marshal(&vm.MessageTx{
		Data: migrationTx,
		To:   to.MarshalPB(),
		From: from.MarshalPB(),
	})
	require.NoError(t, err)

	return messageTx
}
