package plugin

import (
	"github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	regcommon "github.com/loomnetwork/loomchain/registry"
	"github.com/pkg/errors"
)

var (
	// ErrCoinContractNotFound indicates that the Coin contract hasn't been deployed yet.
	ErrCoinContractNotFound = errors.New("[CoinPolicyManager] Coin Contract not found")
)

// CoinPolicyManager implements loomchain.CoinPolicyManager interface
type CoinPolicyManager struct {
	ctx   contract.Context
	state loomchain.State
}

// NewCoinPolicyManager attempts to create an instance of CoinPolicyManager.
func NewCoinPolicyManager(pvm *PluginVM, state loomchain.State) (*CoinPolicyManager, error) {
	caller := loom.RootAddress(pvm.State.Block().ChainID)
	contractAddr, err := pvm.Registry.Resolve("coin")
	if err != nil {
		if err == regcommon.ErrNotFound {
			return nil, ErrCoinContractNotFound
		}
		return nil, err
	}
	readOnly := false
	ctx := contract.WrapPluginContext(pvm.CreateContractContext(caller, contractAddr, readOnly))
	return &CoinPolicyManager{
		ctx:   ctx,
		state: state,
	}, nil
}

// ApplyPolicy will apply the current economic policy within the coin contract
func (c *CoinPolicyManager) ApplyPolicy() error {
	return coin.Mint(c.ctx)
}
