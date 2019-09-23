package middleware

import (
	"context"
	"testing"

	"github.com/loomnetwork/go-loom"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/loomnetwork/loomchain/auth/keys"
	dw "github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	"github.com/loomnetwork/loomchain/features"
	appstate "github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
)

var (
	owner = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	guest = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestDeployerWhitelistMiddleware(t *testing.T) {
	state := appstate.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil, nil)
	state.SetFeature(features.DeployerWhitelistFeature, true)

	txSignedPlugin := mockSignedTx(t, uint64(1), deployId, vm.VMType_PLUGIN, contract)
	txSignedEVM := mockSignedTx(t, uint64(2), deployId, vm.VMType_EVM, contract)
	txSignedMigration := mockSignedTx(t, uint64(3), migrationId, vm.VMType_EVM, contract)
	//init contract
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	dwAddr := fakeCtx.CreateContract(dw.Contract)
	contractContext := contractpb.WrapPluginContext(fakeCtx.WithAddress(dwAddr))

	dwContract := &dw.DeployerWhitelist{}
	require.NoError(t, dwContract.Init(contractContext, &dwtypes.InitRequest{
		Owner: owner.MarshalPB(),
	}))

	guestCtx := context.WithValue(state.Context(), keys.ContextKeyOrigin, guest)
	ownerCtx := context.WithValue(state.Context(), keys.ContextKeyOrigin, owner)

	dwMiddleware, err := NewDeployerWhitelistMiddleware(
		func(_ appstate.State) (contractpb.Context, error) {
			return contractContext, nil
		},
	)
	require.NoError(t, err)

	// unauthorized deployer (DeployTx Plugin)
	_, err = throttleMiddlewareHandler(dwMiddleware, state, txSignedPlugin, guestCtx)
	require.Equal(t, ErrNotAuthorized, err)
	// unauthorized deployer (DeployTx EVM)
	_, err = throttleMiddlewareHandler(dwMiddleware, state, txSignedEVM, guestCtx)
	require.Equal(t, ErrNotAuthorized, err)
	// unauthorized deployer (MigrationTx)
	_, err = throttleMiddlewareHandler(dwMiddleware, state, txSignedMigration, guestCtx)
	require.Equal(t, ErrNotAuthorized, err)

	// authorized deployer
	_, err = throttleMiddlewareHandler(dwMiddleware, state, txSignedPlugin, ownerCtx)
	require.NoError(t, err)
	_, err = throttleMiddlewareHandler(dwMiddleware, state, txSignedEVM, ownerCtx)
	require.NoError(t, err)
	_, err = throttleMiddlewareHandler(dwMiddleware, state, txSignedMigration, ownerCtx)
	require.NoError(t, err)
}
