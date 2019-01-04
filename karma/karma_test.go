package karma

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
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
		{Name: karma.DeployToken, Reward:3, Target:ktypes.KarmaSourceTarget_DEPLOY},
	}

	users = []*ktypes.KarmaAddressSource{
		{ User: user1, Sources: []*ktypes.KarmaSource{{ Name: karma.DeployToken, Count: 104}}},
		{ User: user2, Sources: []*ktypes.KarmaSource{{ Name:karma.DeployToken, Count: 104}}},
	}
)

func TestKarma(t *testing.T) {
	karmaInit := ktypes.KarmaInitRequest{
		Sources: deploySource,
		Users:   users,
		Upkeep:  &ktypes.KarmaUpkeepParams{
			Cost:   10,
			Source: karma.DeployToken,
			Period: period,
		},
	}
	state := mockStateWithKarma(t, karmaInit)

	kh := NewKarmaHandler(factory.RegistryV2, true)
	//createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
	//require.NoError(t, err)
	//reg := createRegistry(state)

	require.NoError(t, kh.Upkeep(state))
	require.Equal(t, int64(104), getKarma(t, state, *user1, karma.DeployToken))

	contract1 := mockDeployEvmContract(t, state, addr1, 1)
	require.True(t, isActive(t, state, contract1))

	state3600 := common.MockStateAt(state, period+1)
	require.NoError(t, kh.Upkeep(state3600))
	require.Equal(t, int64(94), getKarma(t, state3600, *user1, karma.DeployToken))

	contract2 := mockDeployEvmContract(t, state3600, addr1, 2)
	contract3 := mockDeployEvmContract(t, state3600, addr1, 3)
	contract4 := mockDeployEvmContract(t, state3600, addr1, 4)
	contractAddr2 := mockDeployEvmContract(t, state3600, addr2, 1)

	state7200 := common.MockStateAt(state, 2*period+1)
	require.NoError(t, kh.Upkeep(state7200))
	require.Equal(t, int64(54), getKarma(t, state7200, *user1, karma.DeployToken))
	require.Equal(t, int64(94), getKarma(t, state7200, *user2, karma.DeployToken))

	contract5 := mockDeployEvmContract(t, state7200, addr1, 5)
	contract6 := mockDeployEvmContract(t, state7200, addr1, 6)

	require.True(t, isActive(t, state7200, contract1))
	require.True(t, isActive(t, state7200, contract2))
	require.True(t, isActive(t, state7200, contract3))
	require.True(t, isActive(t, state7200, contract4))
	require.True(t, isActive(t, state7200, contract5))
	require.True(t, isActive(t, state7200, contract6))
	require.True(t, isActive(t, state7200, contractAddr2))

	state10800 := common.MockStateAt(state, 3*period+1)
	require.NoError(t, kh.Upkeep(state10800))
	require.Equal(t, int64(4), getKarma(t, state10800, *user1, karma.DeployToken))
	require.Equal(t, int64(84), getKarma(t, state10800, *user2, karma.DeployToken))

	require.True(t, isActive(t, state10800, contract1))
	require.True(t, isActive(t, state10800, contract2))
	require.True(t, isActive(t, state10800, contract3))
	require.True(t, isActive(t, state10800, contract4))
	require.True(t, isActive(t, state10800, contract5))
	require.False(t, isActive(t, state10800, contract6))
	require.True(t, isActive(t, state10800, contractAddr2))


	state14400 := common.MockStateAt(state, 4*period+1)
	require.NoError(t, kh.Upkeep(state14400))
	require.Equal(t, int64(4), getKarma(t, state14400, *user1, karma.DeployToken))
	require.Equal(t, int64(74), getKarma(t, state14400, *user2, karma.DeployToken))

	require.False(t, isActive(t, state14400, contract1))
	require.False(t, isActive(t, state14400, contract2))
	require.False(t, isActive(t, state14400, contract3))
	require.False(t, isActive(t, state14400, contract4))
	require.False(t, isActive(t, state14400, contract5))
	require.False(t, isActive(t, state14400, contract6))
	require.True(t, isActive(t, state14400, contractAddr2))
}

func mockDeployEvmContract(t *testing.T, state loomchain.State, owner loom.Address, nonce uint64) loom.Address {
	createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
	require.NoError(t, err)
	reg := createRegistry(state)

	contractAddr := plugin.CreateAddress(owner, nonce)
	err = reg.Register("", contractAddr, owner)
	require.NoError(t, err)

	karmaState := getKarmaState(t, state)
	require.NoError(t, karma.AddOwnedContract(karmaState, owner, contractAddr, state.Block().Height, nonce));

	return contractAddr
}

func getKarmaState(t *testing.T, state loomchain.State) loomchain.State {
	createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
	require.NoError(t, err)
	reg := createRegistry(state)

	karmaContractAddress, err := reg.Resolve("karma")
	return loomchain.StateWithPrefix(loom.DataPrefix(karmaContractAddress), state)
}

func getKarma(t *testing.T, state loomchain.State, user types.Address, sourceName string) int64 {
	karmaState := getKarmaState(t, state)

	userStateKey := karma.UserStateKey(&user)
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

func isActive(t *testing.T, state loomchain.State, contract loom.Address) bool {
	return getKarmaState(t, state).Has(karma.ContractActiveRecordKey(contract))
}