package karma

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/db"
)

const (
	period = 3600
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")

	types_addr1 = addr1.MarshalPB()
	types_addr2 = addr2.MarshalPB()

	user1   = types_addr1
	user2   = types_addr2

	deploySource = []*ktypes.KarmaSourceReward{
		{karma.DeployToken, 1, ktypes.KarmaSourceTarget_DEPLOY},
	}

	users = []*ktypes.KarmaAddressSource{
		{user1, []*ktypes.KarmaSource{{karma.DeployToken, 10}}},
		{user2, []*ktypes.KarmaSource{{karma.DeployToken, 10}}},
	}
)

func TestKarma(t *testing.T) {
	karmaInit := ktypes.KarmaInitRequest{
		Sources: deploySource,
		Users:   users,
		Upkeep:  &ktypes.KarmaUpkeepParmas{
			Cost:   1,
			Source: karma.DeployToken,
			Period: period,
		},
	}
	state := mockStateWithKarma(t, karmaInit)

	kh := NewKarmaHandler(registry.RegistryV2, true)
	createRegistry, err := factory.NewRegistryFactory(registry.RegistryV2)
	require.NoError(t, err)
	reg := createRegistry(state)

	require.NoError(t, kh.Upkeep(state))
	require.Equal(t, int64(10), getKarma(t, state, *user1, karma.DeployToken))

	contract1 := mockDeployEvmContract(t, state, addr1, 1)
	require.True(t, reg.IsActive(contract1))

	state3600 := common.MockStateAt(state, period+1)
	require.NoError(t, kh.Upkeep(state3600))
	require.Equal(t, int64(9), getKarma(t, state, *user1, karma.DeployToken))

	contract2 := mockDeployEvmContract(t, state, addr1, 2)
	contract3 := mockDeployEvmContract(t, state, addr1, 3)
	contract4 := mockDeployEvmContract(t, state, addr1, 4)
	contract5 := mockDeployEvmContract(t, state, addr1, 5)
	contract6 := mockDeployEvmContract(t, state, addr2, 1)

	state7200 := common.MockStateAt(state, 2*period+1)
	require.NoError(t, kh.Upkeep(state7200))
	require.Equal(t, int64(4), getKarma(t, state, *user1, karma.DeployToken))
	require.Equal(t, int64(9), getKarma(t, state, *user2, karma.DeployToken))


	require.True(t, reg.IsActive(contract1))
	require.True(t, reg.IsActive(contract2))
	require.True(t, reg.IsActive(contract3))
	require.True(t, reg.IsActive(contract4))
	require.True(t, reg.IsActive(contract5))
	require.True(t, reg.IsActive(contract6))
	state10800 := common.MockStateAt(state, 3*period+1)
	require.NoError(t, kh.Upkeep(state10800))
	require.Equal(t, int64(0), getKarma(t, state, *user1, karma.DeployToken))
	require.Equal(t, int64(8), getKarma(t, state, *user2, karma.DeployToken))

	fmt.Println("contract1 active ", reg.IsActive(contract1))
	fmt.Println("contract2 active ", reg.IsActive(contract2))
	fmt.Println("contract3 active ", reg.IsActive(contract3))
	fmt.Println("contract4 active ", reg.IsActive(contract4))
	fmt.Println("contract5 active ", reg.IsActive(contract5))
	fmt.Println("contract6 active ", reg.IsActive(contract6))

	require.True(t, reg.IsActive(contract1))
	require.False(t, reg.IsActive(contract2))
	require.True(t, reg.IsActive(contract3))
	require.True(t, reg.IsActive(contract4))
	require.True(t, reg.IsActive(contract5))
	require.True(t, reg.IsActive(contract6))
}

func mockDeployEvmContract(t *testing.T, state loomchain.State, owner loom.Address, nonce uint64) loom.Address {
	createRegistry, err := factory.NewRegistryFactory(registry.RegistryV2)
	require.NoError(t, err)
	reg := createRegistry(state)

	contractAddr := plugin.CreateAddress(owner, nonce)

	err = reg.Register("", contractAddr, owner)
	require.NoError(t, err)

	return contractAddr
}

func getKarma(t *testing.T, state loomchain.State, user types.Address, sourceName string) int64 {
	createRegistry, err := factory.NewRegistryFactory(registry.RegistryV2)
	require.NoError(t, err)
	reg := createRegistry(state)

	karmaContractAddress, err := reg.Resolve("karma")
	karmaState := loomchain.StateWithPrefix(loom.DataPrefix(karmaContractAddress), state)

	userStateKey := karma.GetUserStateKey(&user)
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

func mockStateWithKarma(t *testing.T,  karmaInit ktypes.KarmaInitRequest) loomchain.State {
	appDb, err := db.NewGoLevelDB("mockAppDBforKarmaUnitTest", ".")
	appStore, err := store.NewIAVLStore(appDb, 1)
	header := abci.Header{}
	header.Height = int64(1)
	state := loomchain.NewStoreState(context.Background(), appStore, header,nil)

	paramCode, err := json.Marshal(karmaInit)
	require.NoError(t, err)
	initCode, err := LoadContractCode("karma:1.0.0", paramCode)
	require.NoError(t, err)
	callerAddr := plugin.CreateAddress(loom.RootAddress("chain"), uint64(0))

	vmManager := vm.NewManager()
	createRegistry, err := factory.NewRegistryFactory(registry.RegistryV2)
	reg := createRegistry(state)
	require.NoError(t, err)
	loader := plugin.NewStaticLoader(karma.Contract)
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