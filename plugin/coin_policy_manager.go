package plugin

import (
	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/config"
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
	cfg   *config.Config
}

type Policy = ctypes.Policy

func NewNoopCoinPolicyManager() *CoinPolicyManager {
	var manager *CoinPolicyManager
	return manager
}

// NewCoinPolicyManager attempts to create an instance of CoinPolicyManager.
func NewCoinPolicyManager(pvm *PluginVM, state loomchain.State, cfg *config.Config) (*CoinPolicyManager, error) {
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
		cfg:   cfg,
	}, nil
}

//MintCoins method of coin_deflation_Manager will be called from Block
func (c *CoinPolicyManager) MintCoins() error {
	err := coin.Mint(c.ctx)
	if err != nil {
		return err
	}
	return nil
}

//ModifyMintCoins method of coin_deflation_Manager will be called from Block
func (c *CoinPolicyManager) ModifyDeflationParameter() error {
	if c.cfg.DeflationInfo.Enabled == true {
		addr, err := loom.ParseAddress(c.cfg.DeflationInfo.MintingAccount)
		if err != nil {
			return errors.Wrapf(err, "parsing deploy address %s", c.cfg.DeflationInfo.MintingAccount)
		}
		policy := &Policy{
			DeflationFactorNumerator:   c.cfg.DeflationInfo.DeflationFactorNumerator,
			DeflationFactorDenominator: c.cfg.DeflationInfo.DeflationFactorDenominator,
			BaseMintingAmount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(c.cfg.DeflationInfo.BaseMintingAmount),
			},
			MintingAccount: addr.MarshalPB(),
		}
		err = coin.ModifyMintParameter(c.ctx, policy)
		c.state.SetFeature(loomchain.CoinPolicyModificationFeature, false)
		if err != nil {
			return err
		}
	}
	return nil
}
