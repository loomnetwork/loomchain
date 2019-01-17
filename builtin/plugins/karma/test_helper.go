package karma

import (
    "encoding/json"
    "testing"
    "context"
    ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
    ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
    "github.com/loomnetwork/loomchain/builtin/plugins/coin"
    "github.com/loomnetwork/loomchain/registry"
    abci "github.com/tendermint/tendermint/abci/types"
    "github.com/loomnetwork/go-loom/types"
    "github.com/loomnetwork/go-loom"
    "github.com/loomnetwork/loomchain"
    "github.com/loomnetwork/loomchain/log"
    "github.com/loomnetwork/loomchain/plugin"
    "github.com/loomnetwork/loomchain/registry/factory"
    "github.com/loomnetwork/loomchain/store"
    "github.com/loomnetwork/loomchain/vm"
    "github.com/stretchr/testify/require"
    "github.com/tendermint/tendermint/libs/db"
    "github.com/gogo/protobuf/proto"

)

func GetKarmaAddress(t *testing.T, state loomchain.State) loom.Address {
    createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
    require.NoError(t, err)
    reg := createRegistry(state)

    karmaContractAddress, err := reg.Resolve("karma")
    require.NoError(t, err)
    return karmaContractAddress
}

func GetKarmaState(t *testing.T, state loomchain.State) loomchain.State {
    return loomchain.StateWithPrefix(loom.DataPrefix(GetKarmaAddress(t, state)), state)
}

func GetKarma(t *testing.T, state loomchain.State, user types.Address, sourceName string) int64 {
    karmaState := GetKarmaState(t, state)

    userStateKey := UserStateKey(&user)
    data := karmaState.Get(userStateKey)
    var userState ktypes.KarmaState
    require.NoError(t, proto.Unmarshal(data, &userState))

    for _, source := range userState.SourceStates {
        if source.Name == sourceName {
            return source.Count.Value.Int64()
        }
    }
    return 0
}

func MockStateWithKarmaAndCoin(t *testing.T,  karmaInit ktypes.KarmaInitRequest, coinInit ctypes.InitRequest) (loomchain.State, registry.Registry, vm.VM) {
    appDb, err := db.NewGoLevelDB("mockAppDBforKarmaUnitTest", ".")
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

    karmaCode, err := json.Marshal(karmaInit)
    require.NoError(t, err)
    karmaInitCode, err := LoadContractCode("karma:1.0.0", karmaCode)
    require.NoError(t, err)
    callerAddr := plugin.CreateAddress(loom.RootAddress("chain"), uint64(0))
    _, karmaAddr, err := pluginVm.Create(callerAddr, karmaInitCode, loom.NewBigUIntFromInt(0))
    require.NoError(t, err)
    err = reg.Register("karma", karmaAddr, karmaAddr)
    require.NoError(t, err)


    coinCode, err := json.Marshal(coinInit)
    require.NoError(t, err)
    coinInitCode, err := LoadContractCode("coin:1.0.0", coinCode)
    require.NoError(t, err)
    callerAddr = plugin.CreateAddress(loom.RootAddress("chain"), uint64(1))
    _, coinAddr, err := pluginVm.Create(callerAddr, coinInitCode, loom.NewBigUIntFromInt(0))
    require.NoError(t, err)
    err = reg.Register("coin", coinAddr, coinAddr)
    require.NoError(t, err)

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
