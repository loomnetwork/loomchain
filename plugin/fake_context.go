// +build evm

package plugin

import (
	"context"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	levm "github.com/loomnetwork/loomchain/evm"
	abci "github.com/tendermint/tendermint/abci/types"
)

// Contract context for tests that need both Go & EVM contracts.
type FakeContextWithEVM struct {
	*plugin.FakeContext
	State loomchain.State
}

func CreateFakeContextWithEVM(caller, address loom.Address) *FakeContextWithEVM {
	block := abci.Header{
		ChainID: "chain",
		Height:  int64(34),
		Time:    int64(123456789),
	}
	ctx := plugin.CreateFakeContext(caller, address).WithBlock(
		types.BlockHeader{
			ChainID: block.ChainID,
			Height:  block.Height,
			Time:    block.Time,
		},
	)
	state := loomchain.NewStoreState(context.Background(), ctx, block)
	return &FakeContextWithEVM{
		FakeContext: ctx,
		State:       state,
	}
}

func (c *FakeContextWithEVM) WithBlock(header loom.BlockHeader) *FakeContextWithEVM {
	return &FakeContextWithEVM{
		FakeContext: c.FakeContext.WithBlock(header),
		State:       c.State,
	}
}

func (c *FakeContextWithEVM) WithSender(caller loom.Address) *FakeContextWithEVM {
	return &FakeContextWithEVM{
		FakeContext: c.FakeContext.WithSender(caller),
		State:       c.State,
	}
}

func (c *FakeContextWithEVM) WithAddress(addr loom.Address) *FakeContextWithEVM {
	return &FakeContextWithEVM{
		FakeContext: c.FakeContext.WithAddress(addr),
		State:       c.State,
	}
}

func (c *FakeContextWithEVM) CallEVM(addr loom.Address, input []byte) ([]byte, error) {
	vm := levm.NewLoomVm(c.State, nil)
	return vm.Call(c.ContractAddress(), addr, input)
}

func (c *FakeContextWithEVM) StaticCallEVM(addr loom.Address, input []byte) ([]byte, error) {
	vm := levm.NewLoomVm(c.State, nil)
	return vm.StaticCall(c.ContractAddress(), addr, input)
}
