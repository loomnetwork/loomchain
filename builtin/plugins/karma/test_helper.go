package karma

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	ktypes "github.com/loomnetwork/go-loom/builtin/types/karma"
	glplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
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

// TODO: This duplicates a lot of the contract loading & deployment code, it should just use the
//       mock context, or if that's not sufficient the contract loading code may need to be
//       refactored to make it possible to eliminate these helpers.

type MockContractDetails struct {
	Name     string
	Version  string
	Init     interface{}
	Contract glplugin.Contract
}

func MockStateWithKarmaAndCoinT(t *testing.T, karmaInit *ktypes.KarmaInitRequest, coinInit *ctypes.InitRequest) (loomchain.State, registry.Registry, vm.VM) {
	appDb := db.NewMemDB()
	state, reg, manager, err := MockStateWithContracts(
		appDb,
		MockContractDetails{"karma", "1.0.0", karmaInit, Contract},
		MockContractDetails{"coin", "1.0.0", coinInit, coin.Contract},
	)

	require.NoError(t, err)
	pluginVm, err := manager.InitVM(vm.VMType_PLUGIN, state)
	require.NoError(t, err)
	return state, reg, pluginVm
}

func MockStateWithKarmaAndCoinB(b *testing.B, karmaInit *ktypes.KarmaInitRequest, coinInit *ctypes.InitRequest, appDbName string) (loomchain.State, registry.Registry, vm.VM) {
	appDb := db.NewMemDB()
	state, reg, manager, err := MockStateWithContracts(
		appDb,
		MockContractDetails{"karma", "1.0.0", karmaInit, Contract},
		MockContractDetails{"coin", "1.0.0", coinInit, coin.Contract},
	)

	require.NoError(b, err)
	pluginVm, err := manager.InitVM(vm.VMType_PLUGIN, state)
	require.NoError(b, err)
	return state, reg, pluginVm
}

func MockStateWithContracts(appDb db.DB, contracts ...MockContractDetails) (loomchain.State, registry.Registry, *vm.Manager, error) {
	appStore, err := store.NewIAVLStore(appDb, 0, 0, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	header := abci.Header{}
	header.Height = int64(1)
	state := loomchain.NewStoreState(context.Background(), appStore, header, nil, nil)

	vmManager := vm.NewManager()
	createRegistry, err := factory.NewRegistryFactory(factory.RegistryV2)
	reg := createRegistry(state)
	if err != nil {
		return nil, nil, nil, err
	}

	loadList := []glplugin.Contract{}
	for _, contract := range contracts {
		loadList = append(loadList, contract.Contract)
	}
	loader := &plugin.StaticLoader{Contracts: loadList}

	vmManager.Register(vm.VMType_PLUGIN, func(state loomchain.State) (vm.VM, error) {
		return plugin.NewPluginVM(loader, state, reg, &FakeEventHandler{}, log.Default, nil, nil, nil), nil
	})
	pluginVm, err := vmManager.InitVM(vm.VMType_PLUGIN, state)
	if err != nil {
		return nil, nil, nil, err
	}

	for i, contract := range contracts {
		code, err := json.Marshal(contract.Init)
		if err != nil {
			return nil, nil, nil, err
		}
		initCode, err := LoadContractCode(contract.Name+":"+contract.Version, code)
		if err != nil {
			return nil, nil, nil, err
		}
		callerAddr := plugin.CreateAddress(loom.RootAddress("chain"), uint64(i))
		_, addr, err := pluginVm.Create(callerAddr, initCode, loom.NewBigUIntFromInt(0))
		if err != nil {
			return nil, nil, nil, err
		}
		err = reg.Register(contract.Name, addr, addr)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return state, reg, vmManager, nil
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

func MockDeployEvmContract(t *testing.T, karamContractCtx contractpb.Context, owner loom.Address, nonce uint64) loom.Address {
	contractAddr := plugin.CreateAddress(owner, nonce)
	require.NoError(t, AddOwnedContract(karamContractCtx, owner, contractAddr))
	return contractAddr
}
