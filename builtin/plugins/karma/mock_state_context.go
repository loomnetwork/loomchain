package karma

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain/vm"

	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/registry"
)

type FakeStateContext struct {
	plugin.FakeContext
	state    loomchain.State
	registry registry.Registry
	VM       vm.VM
}

func CreateFakeStateContext(state loomchain.State, reg registry.Registry, caller, address loom.Address, pluginVm vm.VM) *FakeStateContext {
	fakeContext := plugin.CreateFakeContext(caller, address)
	return &FakeStateContext{
		FakeContext: *fakeContext.WithSender(caller),
		state:       state.WithPrefix(loom.DataPrefix(address)),
		registry:    reg,
		VM:          pluginVm,
	}
}

func (c *FakeStateContext) Range(prefix []byte) plugin.RangeData {
	return c.state.Range(prefix)
}

func (c *FakeStateContext) Get(key []byte) []byte {
	return c.state.Get(key)
}

func (c *FakeStateContext) Has(key []byte) bool {
	return c.state.Has(key)
}

func (c *FakeStateContext) Set(key []byte, value []byte) {
	c.state.Set(key, value)
}

func (c *FakeStateContext) Delete(key []byte) {
	c.state.Delete(key)
}

func (c *FakeStateContext) Resolve(name string) (loom.Address, error) {
	return c.registry.Resolve(name)
}

func (c *FakeStateContext) Call(addr loom.Address, input []byte) ([]byte, error) {
	return c.VM.Call(c.FakeContext.ContractAddress(), addr, input, common.BigZero())
}
