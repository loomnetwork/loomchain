package karma

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/loomchain"
)

type FakeStateContext struct {
	plugin.FakeContext
	state 	      loomchain.State
}

func CreateFakeStateContext(state loomchain.State, caller, address loom.Address) *FakeStateContext {
	fakeContext := plugin.CreateFakeContext(caller, address)
	return &FakeStateContext{
		 FakeContext:	*fakeContext,
		 state:     	loomchain.StateWithPrefix(loom.DataPrefix(address), state),
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

