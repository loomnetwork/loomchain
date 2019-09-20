package migrations

import (
	"github.com/pkg/errors"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	genesiscfg "github.com/loomnetwork/loomchain/config/genesis"
	"github.com/loomnetwork/loomchain/core"
	"github.com/loomnetwork/loomchain/plugin"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	appstate "github.com/loomnetwork/loomchain/state"
	"github.com/loomnetwork/loomchain/vm"
)

var (
	ErrLoaderNotFound = errors.New("loader not found")
)

// MigrationContext is available within migration functions, it can be used to deploy new contracts,
// and to call Go & EVM contracts.
type MigrationContext struct {
	manager        *vm.Manager
	createRegistry registry.RegistryFactoryFunc
	caller         loom.Address
	state          appstate.State
	codeLoaders    map[string]core.ContractCodeLoader
}

func NewMigrationContext(
	manager *vm.Manager,
	createRegistry registry.RegistryFactoryFunc,
	state appstate.State,
	caller loom.Address,
) *MigrationContext {
	return &MigrationContext{
		manager:        manager,
		createRegistry: createRegistry,
		state:          state,
		caller:         caller,
		codeLoaders:    core.GetDefaultCodeLoaders(),
	}
}

// State returns the app state.
func (mc *MigrationContext) State() appstate.State {
	return mc.state
}

// DeployContract deploys a Go contract and returns its address.
func (mc *MigrationContext) DeployContract(contractCfg *genesiscfg.ContractConfig) (loom.Address, error) {
	vmType := contractCfg.VMType()
	vm, err := mc.manager.InitVM(vmType, mc.state)
	if err != nil {
		return loom.Address{}, err
	}

	loader, found := mc.codeLoaders[contractCfg.Format]
	if !found {
		return loom.Address{}, errors.Wrapf(ErrLoaderNotFound, "contract format: %s", contractCfg.Format)
	}
	initCode, err := loader.LoadContractCode(
		contractCfg.Location,
		contractCfg.Init,
	)
	if err != nil {
		return loom.Address{}, err
	}
	_, addr, err := vm.Create(mc.caller, initCode, loom.NewBigUIntFromInt(0))
	if err != nil {
		return loom.Address{}, err
	}

	reg := mc.createRegistry(mc.state)
	if err := reg.Register(contractCfg.Name, addr, mc.caller); err != nil {

		return loom.Address{}, err
	}

	return addr, nil
}

// ContractContext returns the context for the Go contract with the given name.
// The returned context can be used to call the named contract.
func (mc *MigrationContext) ContractContext(contractName string) (contractpb.Context, error) {
	reg := mc.createRegistry(mc.state)
	contractAddr, err := reg.Resolve(contractName)
	if err != nil {
		return nil, err
	}

	vm, err := mc.manager.InitVM(vm.VMType_PLUGIN, mc.state)
	if err != nil {
		return nil, err
	}

	pluginVM := vm.(*plugin.PluginVM) // Ugh
	readOnly := false
	return contractpb.WrapPluginContext(pluginVM.CreateContractContext(mc.caller, contractAddr, readOnly)), nil
}
