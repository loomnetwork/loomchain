package throttle

import (
	"context"
	"testing"

	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	sourcesDeploy = []*ktypes.KarmaSourceReward{
		{Name: "award1", Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
		{Name: "award2", Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
		{Name: "award3", Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
		{Name: "sms", Reward: 1, Target: ktypes.KarmaSourceTarget_CALL},
		{Name: karma.CoinDeployToken, Reward: 1, Target: ktypes.KarmaSourceTarget_DEPLOY},
	}

	sourceStatesDeploy = []*ktypes.KarmaSource{
		{Name: "sms", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)}},
		{Name: "award1", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)}},
		{Name: karma.CoinDeployToken, Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)}},
	}
)

func TestKarmaMiddleWare(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil, nil, nil)

	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	karmaAddr := fakeCtx.CreateContract(karma.Contract)
	contractContext := contractpb.WrapPluginContext(fakeCtx.WithAddress(karmaAddr))

	// Init the karma contract
	karmaContract := &karma.Karma{}
	require.NoError(t, karmaContract.Init(contractContext, &ktypes.KarmaInitRequest{
		Sources: sourcesDeploy,
	}))

	// This can also be done on init, but more concise this way
	require.NoError(t, karma.AddKarma(contractContext, origin, sourceStatesDeploy))

	ctx := context.WithValue(state.Context(), auth.ContextKeyOrigin, origin)

	tmx := GetKarmaMiddleWare(
		true,
		maxCallCount,
		sessionDuration,
		func(state loomchain.State) (contractpb.Context, error) {
			return contractContext, nil
		},
	)

	// call fails as contract is not deployed
	txSigned := mockSignedTx(t, uint64(1), callId, vm.VMType_EVM, contract)
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.Error(t, err)

	// deploy contract
	txSigned = mockSignedTx(t, uint64(2), deployId, vm.VMType_EVM, contract)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// call now works
	txSigned = mockSignedTx(t, uint64(3), callId, vm.VMType_EVM, contract)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// deactivate contract
	record, err := karma.GetContractRecord(contractContext, contract)
	require.NoError(t, err)
	require.NoError(t, karma.DeactivateContract(contractContext, record))

	// call now fails
	txSigned = mockSignedTx(t, uint64(4), callId, vm.VMType_EVM, contract)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.Error(t, err)
}

func TestMinKarmaToDeploy(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil, nil, nil)

	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	karmaAddr := fakeCtx.CreateContract(karma.Contract)
	contractContext := contractpb.WrapPluginContext(fakeCtx.WithAddress(karmaAddr))

	// Init the karma contract
	karmaContract := &karma.Karma{}
	require.NoError(t, karmaContract.Init(contractContext, &ktypes.KarmaInitRequest{
		Sources: sourcesDeploy,
	}))

	require.NoError(t, karma.SetConfig(contractContext, &ktypes.KarmaConfig{
		MinKarmaToDeploy: 1,
	}))

	require.NoError(t, karma.AddKarma(contractContext, origin, sourceStatesDeploy))

	ctx := context.WithValue(state.Context(), auth.ContextKeyOrigin, origin)

	tmx := GetKarmaMiddleWare(
		true,
		maxCallCount,
		sessionDuration,
		func(state loomchain.State) (contractpb.Context, error) {
			return contractContext, nil
		},
	)

	// deploy contract
	txSigned := mockSignedTx(t, uint64(2), deployId, vm.VMType_EVM, contract)
	_, err := throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	require.NoError(t, karma.SetConfig(contractContext, &ktypes.KarmaConfig{
		MinKarmaToDeploy: 2000,
	}))

	// deploy contract
	txSigned = mockSignedTx(t, uint64(2), deployId, vm.VMType_EVM, contract)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.Error(t, err)
}
