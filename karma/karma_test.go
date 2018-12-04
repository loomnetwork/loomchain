package karma

import (
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma"
	"github.com/loomnetwork/loomchain/log"
	"github.com/loomnetwork/loomchain/plugin"
	"github.com/loomnetwork/loomchain/receipts/common"
	"github.com/loomnetwork/loomchain/registry"
	"github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
)

const (
	period = 3600
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr4 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")

	types_addr1 = addr1.MarshalPB()
	types_addr4 = addr4.MarshalPB()


	oracle  = types_addr1
	user    = types_addr4

	deploySource = []*ktypes.KarmaSourceReward{
		{karma.DeployToken, 1, ktypes.KarmaSourceTarget_DEPLOY},
	}

	sourceStates = []*ktypes.KarmaSource{
		{karma.DeployToken, 1},
	}

	users = []*ktypes.KarmaAddressSource{
		{user, sourceStates},
		{oracle, sourceStates},
	}
)

func TestKarma(t *testing.T) {
	kh := NewKarmaHandler(registry.RegistryV2, true)
	state := mockStateWithKarma(t)
	require.NoError(t, kh.Upkeep(state))

	state3600 := common.MockStateAt(state, period+1)
	require.NoError(t, kh.Upkeep(state3600))

}

func mockStateWithKarma(t *testing.T) loomchain.State {
	state := common.MockState(1)

	karmaInit := ktypes.KarmaInitRequest{
		Sources: deploySource,
		Oracle:  oracle,
		Users:   users,
		Upkeep:  &ktypes.KarmaUpkeepParmas{
			Cost:   1,
			Source: karma.DeployToken,
			Period: period,
		},
	}
	paramCode, err := json.Marshal(karmaInit)
	require.NoError(t, err)
	initCode, err := LoadContractCode("karma:1.0.0", paramCode)
	require.NoError(t, err)
	callerAddr := plugin.CreateAddress(loom.RootAddress("default"), uint64(0))

	vmManager := vm.NewManager()
	createRegistry, err := factory.NewRegistryFactory(registry.RegistryV2)
	reg := createRegistry(state)
	require.NoError(t, err)
	loader := plugin.NewStaticLoader(karma.Contract)
	vmManager.Register(vm.VMType_PLUGIN, func(state loomchain.State) (vm.VM, error) {
		return plugin.NewPluginVM(
			loader,
			state,
			reg,
			nil,
			log.Default,
			nil,
			nil,
			nil,
		), nil
	})
	pluginVm, err := vmManager.InitVM(vm.VMType_PLUGIN, state)
	require.NoError(t, err)

	_, karmaAddr, err := pluginVm.Create(callerAddr, initCode, loom.NewBigUIntFromInt(0))
	require.NoError(t, err)
	err = reg.Register("karma", karmaAddr, karmaAddr)
	require.NoError(t, err)
	return state
}

func LoadContractCode(location string, init json.RawMessage) ([]byte, error) {
	// just verify that it's json
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