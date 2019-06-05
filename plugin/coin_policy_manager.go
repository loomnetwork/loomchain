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
	ErrCoinContractNotFound = errors.New("[CoinDeflationManager] CoinContract contract not found")
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

//MintCoins method of coin_deflation_Manager will be called from Block
func (c *CoinPolicyManager) MintCoins() error {
	if c.state.FeatureEnabled(loomchain.CoinPolicyFeature, false) {
		err := coin.Mint(c.ctx)
		if err != nil {
			return err
		}
	}
	return nil
}
