package throttle

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/registry/factory"
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

	userStateDeploy = ktypes.KarmaState{ //types.BigUInt
		SourceStates:     sourceStatesDeploy,
		DeployKarmaTotal: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1*10 + 1*maxDeployCount)},
		CallKarmaTotal:   &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)},
	}

	userStateMin = ktypes.KarmaState{ //types.BigUInt
		SourceStates: []*ktypes.KarmaSource{
			{Name: "award1", Count: &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)}},
		},
		DeployKarmaTotal: &types.BigUInt{Value: *loom.NewBigUIntFromInt(1*10 + 1*maxDeployCount)},
		CallKarmaTotal:   &types.BigUInt{Value: *loom.NewBigUIntFromInt(10)},
	}
)

func TestKarmaMiddleWare(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil)

	var createRegistry factory.RegistryFactoryFunc
	createRegistry, err := factory.NewRegistryFactory(factory.LatestRegistryVersion)
	require.NoError(t, err)
	registryObject := createRegistry(state)

	contractContext := contractpb.WrapPluginContext(
		goloomplugin.CreateFakeContext(addr1, addr1),
	)
	karmaAddr := contractContext.ContractAddress()
	karmaState := loomchain.StateWithPrefix(loom.DataPrefix(karmaAddr), state)
	require.NoError(t, registryObject.Register("karma", karmaAddr, addr1))

	sourcesB, err := proto.Marshal(&ktypes.KarmaSources{
		Sources: sourcesDeploy,
	})
	require.NoError(t, err)
	karmaState.Set(karma.SourcesKey, sourcesB)

	sourceStatesB, err := proto.Marshal(&userStateDeploy)
	require.NoError(t, err)
	stateKey := karma.UserStateKey(origin.MarshalPB())
	karmaState.Set(stateKey, sourceStatesB)

	ctx := context.WithValue(state.Context(), auth.ContextKeyOrigin, origin)

	tmx := GetKarmaMiddleWare(
		true,
		maxCallCount,
		sessionDuration,
		factory.LatestRegistryVersion,
	)

	// call fails as contract is not deployed
	txSigned := mockSignedTx(t, uint64(1), callId, vm.VMType_EVM, contract)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.Error(t, err)

	// deploy contract
	txSigned = mockSignedTx(t, uint64(2), deployId, vm.VMType_EVM, contract)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// call now works
	txSigned = mockSignedTx(t, uint64(3), callId, vm.VMType_EVM, contract)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	// inactivate contract
	var record ktypes.KarmaContractRecord
	require.NoError(t, proto.Unmarshal(karmaState.Get(karma.ContractRecordKey(contract)), &record))
	require.NoError(t, karma.SetInactive(karmaState, record))

	// call now fails
	txSigned = mockSignedTx(t, uint64(4), callId, vm.VMType_EVM, contract)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.Error(t, err)
}

func TestMinKarmaToDeploy(t *testing.T) {
	log.Setup("debug", "file://-")
	log.Root.With("module", "throttle-middleware")

	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{}, nil)

	var createRegistry factory.RegistryFactoryFunc
	createRegistry, err := factory.NewRegistryFactory(factory.LatestRegistryVersion)
	require.NoError(t, err)
	registryObject := createRegistry(state)

	contractContext := contractpb.WrapPluginContext(
		goloomplugin.CreateFakeContext(addr1, addr1),
	)
	karmaAddr := contractContext.ContractAddress()
	karmaState := loomchain.StateWithPrefix(loom.DataPrefix(karmaAddr), state)
	require.NoError(t, registryObject.Register("karma", karmaAddr, addr1))

	sourcesB, err := proto.Marshal(&ktypes.KarmaSources{
		Sources: sourcesDeploy,
	})
	require.NoError(t, err)
	karmaState.Set(karma.SourcesKey, sourcesB)

	configB, err := proto.Marshal(&ktypes.KarmaConfig{
		MinKarmaToDeploy: 1,
	})
	require.NoError(t, err)
	karmaState.Set(karma.ConfigKey, configB)

	sourceStatesB, err := proto.Marshal(&userStateDeploy)
	require.NoError(t, err)
	stateKey := karma.UserStateKey(origin.MarshalPB())
	karmaState.Set(stateKey, sourceStatesB)

	ctx := context.WithValue(state.Context(), auth.ContextKeyOrigin, origin)

	tmx := GetKarmaMiddleWare(
		true,
		maxCallCount,
		sessionDuration,
		factory.LatestRegistryVersion,
	)

	// deploy contract
	txSigned := mockSignedTx(t, uint64(2), deployId, vm.VMType_EVM, contract)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.NoError(t, err)

	configB, err = proto.Marshal(&ktypes.KarmaConfig{
		MinKarmaToDeploy: 2000,
	})
	require.NoError(t, err)
	karmaState.Set(karma.ConfigKey, configB)

	// deploy contract
	txSigned = mockSignedTx(t, uint64(2), deployId, vm.VMType_EVM, contract)
	_, err = throttleMiddlewareHandler(tmx, state, txSigned, ctx)
	require.Error(t, err)
}
