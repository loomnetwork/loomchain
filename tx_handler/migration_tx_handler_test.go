package tx_handler

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/go-loom/vm"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/migrations"
	appstate "github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
)

func TestMigrationTxHandler(t *testing.T) {
	origin := loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	state := appstate.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil, nil)
	state.SetFeature(features.MigrationTxFeature, true)

	ctx := context.WithValue(state.Context(), auth.ContextKeyOrigin, origin)
	s := state.WithContext(ctx)

	migrationTx1 := mockMessageTx(t, uint32(1), origin, origin, []byte{})

	migrationFuncs := map[int32]MigrationFunc{
		1: func(ctx *migrations.MigrationContext, parameters []byte) error { return nil },
		2: func(ctx *migrations.MigrationContext, parameters []byte) error { return nil },
		3: func(ctx *migrations.MigrationContext, parameters []byte) error {
			var addr types.Address
			if err := proto.Unmarshal(parameters, &addr); err != nil {
				return err
			}
			if origin.Compare(loom.UnmarshalAddressPB(&addr)) != 0 {
				return errors.New("Invalid input message")
			}
			return nil
		},
	}

	migrationTxHandler := &MigrationTxHandler{
		Manager:        nil,
		CreateRegistry: nil,
		Migrations:     migrationFuncs,
	}

	state.SetFeature(features.MigrationTxFeature, true)
	state.SetFeature(features.MigrationFeaturePrefix+"1", true)
	_, err := migrationTxHandler.ProcessTx(s, migrationTx1, false)
	require.NoError(t, err)

	_, err = migrationTxHandler.ProcessTx(s, migrationTx1, false)
	require.Error(t, err)

	addressBytes, err := proto.Marshal(origin.MarshalPB())
	require.NoError(t, err)

	migrationTx2 := mockMessageTx(t, uint32(2), origin, origin, []byte{})
	migrationTx3 := mockMessageTx(t, uint32(3), origin, origin, addressBytes)
	migrationTx4 := mockMessageTx(t, uint32(4), origin, origin, []byte{})

	state.SetFeature(features.MigrationTxFeature, true)
	state.SetFeature(features.MigrationFeaturePrefix+"3", true)
	_, err = migrationTxHandler.ProcessTx(s, migrationTx3, false)
	require.NoError(t, err)

	//Expect an error if migrationtx feature is not enabled
	state.SetFeature(features.MigrationTxFeature, false)
	_, err = migrationTxHandler.ProcessTx(s, migrationTx2, false)
	require.Error(t, err)

	//Expect an error if migration id is not found
	state.SetFeature(features.MigrationTxFeature, true)
	_, err = migrationTxHandler.ProcessTx(s, migrationTx4, false)
	require.Error(t, err)

	//Expect an error if migration feature is not enabled
	_, err = migrationTxHandler.ProcessTx(s, migrationTx2, false)
	require.Error(t, err)

	state.SetFeature(features.MigrationTxFeature, true)
	state.SetFeature(features.MigrationFeaturePrefix+"2", true)
	_, err = migrationTxHandler.ProcessTx(s, migrationTx2, false)
	require.NoError(t, err)

}

func mockMessageTx(t *testing.T, id uint32, to loom.Address, from loom.Address, input []byte) []byte {
	var messageTx []byte

	migrationTx, err := proto.Marshal(&vm.MigrationTx{
		ID:    id,
		Input: input,
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
