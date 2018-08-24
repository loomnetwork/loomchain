package plugin

import (
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain/builtin/plugins/ethcoin"
	"github.com/loomnetwork/loomchain/evm"
)

// AccountBalanceManager implements the evm.AccountBalanceManager interface using the built-in
// ethcoin contract.
type AccountBalanceManager struct {
	// ethcoin contract context
	ctx  contract.Context
	sctx contract.StaticContext
}

func NewAccountBalanceManager(ctx plugin.Context) *AccountBalanceManager {
	return &AccountBalanceManager{
		ctx:  contract.WrapPluginContext(ctx),
		sctx: contract.WrapPluginStaticContext(ctx),
	}
}

func (m *AccountBalanceManager) GetBalance(addr loom.Address) (*loom.BigUInt, error) {
	return ethcoin.BalanceOf(m.sctx, addr)
}

func (m *AccountBalanceManager) AddBalance(addr loom.Address, amount *loom.BigUInt) error  {
	return ethcoin.AddBalance(m.ctx, addr, amount)
}

func (m *AccountBalanceManager) SubBalance(addr loom.Address, amount *loom.BigUInt) error {
	return ethcoin.SubBalance(m.ctx,  addr, amount)
}

func (m *AccountBalanceManager) SetBalance(addr loom.Address, amount *loom.BigUInt) error {
	return ethcoin.SetBalance(m.ctx,  addr, amount)
}

func (m *AccountBalanceManager) Transfer(from, to loom.Address, amount *loom.BigUInt) error {
	return ethcoin.Transfer(m.ctx, from, to, amount)
}

type NewAccountBalanceManagerFactoryFunc func(*PluginVM) (evm.AccountBalanceManagerFactoryFunc, error)

func NewAccountBalanceManagerFactory(pvm *PluginVM) (evm.AccountBalanceManagerFactoryFunc, error) {
	ethCoinAddr, err := pvm.Registry.Resolve("ethcoin")
	if err != nil {
		return nil, err
	}
	return func(readOnly bool) evm.AccountBalanceManager {
		caller := loom.RootAddress(pvm.State.Block().ChainID)
		ctx := pvm.createContractContext(caller, ethCoinAddr, readOnly)
		return NewAccountBalanceManager(ctx)
	}, nil
}
