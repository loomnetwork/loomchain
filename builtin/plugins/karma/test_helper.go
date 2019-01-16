package karma

import (
    "encoding/json"
    "testing"
    "context"
    ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
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

func MockDeployEvmContract(t *testing.T, state loomchain.State, owner loom.Address, nonce uint64) loom.Address {
    createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
    require.NoError(t, err)
    reg := createRegistry(state)

    contractAddr := plugin.CreateAddress(owner, nonce)
    err = reg.Register("", contractAddr, owner)
    require.NoError(t, err)

    karmaState := GetKarmaState(t, state)
    require.NoError(t, AddOwnedContract(karmaState, owner, contractAddr, state.Block().Height, nonce));

    return contractAddr
}

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
            return source.Count
        }
    }
    return 0
}

func MockStateWithKarma(t *testing.T,  karmaInit ktypes.KarmaInitRequest) loomchain.State {
    appDb, err := db.NewGoLevelDB("mockAppDBforKarmaUnitTest", ".")
    appStore, err := store.NewIAVLStore(appDb, 0,0)
    header := abci.Header{}
    header.Height = int64(1)
    state := loomchain.NewStoreState(context.Background(), appStore, header,nil)

    paramCode, err := json.Marshal(karmaInit)
    require.NoError(t, err)
    initCode, err := LoadContractCode("karma:1.0.0", paramCode)
    require.NoError(t, err)
    callerAddr := plugin.CreateAddress(loom.RootAddress("chain"), uint64(0))

    vmManager := vm.NewManager()
    createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
    reg := createRegistry(state)
    require.NoError(t, err)
    loader := plugin.NewStaticLoader(Contract)
    vmManager.Register(vm.VMType_PLUGIN, func(state loomchain.State) (vm.VM, error) {
        return plugin.NewPluginVM(loader, state, reg,nil, log.Default,nil,nil,nil,), nil
    })
    pluginVm, err := vmManager.InitVM(vm.VMType_PLUGIN, state)
    require.NoError(t, err)

    _, karmaAddr, err := pluginVm.Create(callerAddr, initCode, loom.NewBigUIntFromInt(0))
    require.NoError(t, err)
    err = reg.Register("karma", karmaAddr, karmaAddr)
    require.NoError(t, err)
    return state
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

func IsActive(t *testing.T, state loomchain.State, contract loom.Address) bool {
    return GetKarmaState(t, state).Has(ContractActiveRecordKey(contract))
}
