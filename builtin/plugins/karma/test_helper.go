package karma

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
)


func MockStateWithKarmaAndCoin(t *testing.T,  karmaInit *ktypes.KarmaInitRequest, coinInit *ctypes.InitRequest, appDbName string) (loomchain.State, registry.Registry, vm.VM) {
	appDb, err := db.NewGoLevelDB(appDbName, ".")
	appStore, err := store.NewIAVLStore(appDb, 0,0)
	header := abci.Header{}
	header.Height = int64(1)
	state := loomchain.NewStoreState(context.Background(), appStore, header,nil)

	vmManager := vm.NewManager()
	createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
	reg := createRegistry(state)
	require.NoError(t, err)
	loader := plugin.NewStaticLoader(Contract, coin.Contract)
	vmManager.Register(vm.VMType_PLUGIN, func(state loomchain.State) (vm.VM, error) {
		return plugin.NewPluginVM(loader, state, reg,nil, log.Default,nil,nil,nil,), nil
	})
	pluginVm, err := vmManager.InitVM(vm.VMType_PLUGIN, state)
	require.NoError(t, err)

	if karmaInit != nil {
		karmaCode, err := json.Marshal(karmaInit)
		require.NoError(t, err)
		karmaInitCode, err := LoadContractCode("karma:1.0.0", karmaCode)
		require.NoError(t, err)
		callerAddr := plugin.CreateAddress(loom.RootAddress("chain"), uint64(0))
		_, karmaAddr, err := pluginVm.Create(callerAddr, karmaInitCode, loom.NewBigUIntFromInt(0))
		require.NoError(t, err)
		err = reg.Register("karma", karmaAddr, karmaAddr)
		require.NoError(t, err)
	}

	if coinInit != nil {
		coinCode, err := json.Marshal(coinInit)
		require.NoError(t, err)
		coinInitCode, err := LoadContractCode("coin:1.0.0", coinCode)
		require.NoError(t, err)
		callerAddr := plugin.CreateAddress(loom.RootAddress("chain"), uint64(1))
		_, coinAddr, err := pluginVm.Create(callerAddr, coinInitCode, loom.NewBigUIntFromInt(0))
		require.NoError(t, err)
		err = reg.Register("coin", coinAddr, coinAddr)
		require.NoError(t, err)
	}
	return state, reg, pluginVm
}

// copied from PluginCodeLoader.LoadContractCode maybe move PluginCodeLoader to separate package
func LoadContractCode(location string, init json.RawMessage) ([]byte, error) {
    body, err := init.MarshalJSON()
    if err != nil {
        return nil, err
    }

    req := &plugin.Request{
        ContentType: plugin.EncodingType_JSON,
        Body:        body,
    }

    input, err := proto.Marshal(req)
    if err != nil {
        return nil, err
    }

    pluginCode := &plugin.PluginCode{
        Name:  location,
        Input: input,
    }
    return proto.Marshal(pluginCode)
}

func MockDeployEvmContract(t *testing.T, state loomchain.State, owner loom.Address, nonce uint64, reg registry.Registry) loom.Address {
	contractAddr := plugin.CreateAddress(owner, nonce)
	err := reg.Register("", contractAddr, owner)
	require.NoError(t, err)

	karmaState := GetKarmaState(t, state, reg)
	require.NoError(t, AddOwnedContract(karmaState, owner, contractAddr, state.Block().Height, nonce));

	return contractAddr
}

func GetKarmaState(t *testing.T, state loomchain.State, reg registry.Registry) loomchain.State {
	karmaAddr, err := reg.Resolve("karma")
	require.NoError(t, err)
	return loomchain.StateWithPrefix(loom.DataPrefix(karmaAddr), state)
}