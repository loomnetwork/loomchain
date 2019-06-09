package plugin

import (
	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
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
	ctx    contract.Context
	state  loomchain.State
	policy *Policy
}

type Policy = ctypes.Policy

func NewNoopCoinPolicyManager() *CoinPolicyManager {
	var manager *CoinPolicyManager
	return manager
}

// NewCoinPolicyManager attempts to create an instance of CoinPolicyManager.
func NewCoinPolicyManager(pvm *PluginVM, state loomchain.State, deflationFactorNumerator uint64,
	deflationFactorDenominator uint64, baseMintingAmount int64, mintingAddress *types.Address) (*CoinPolicyManager,
	error) {
	caller := loom.RootAddress(pvm.State.Block().ChainID)
	contractAddr, err := pvm.Registry.Resolve("coin")
	if err != nil {
		if err == regcommon.ErrNotFound {
			return nil, ErrCoinContractNotFound
		}
		return nil, err
	}
	policy := &Policy{
		DeflationFactorNumerator:   deflationFactorNumerator,
		DeflationFactorDenominator: deflationFactorDenominator,
		BaseMintingAmount: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(baseMintingAmount),
		},
		MintingAccount: mintingAddress,
	}
	readOnly := false
	ctx := contract.WrapPluginContext(pvm.CreateContractContext(caller, contractAddr, readOnly))
	return &CoinPolicyManager{
		ctx:    ctx,
		state:  state,
		policy: policy,
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
	err := coin.ModifyMintParameter(c.ctx, c.policy)
	if err != nil {
		return err
	}
	c.state.SetFeature(loomchain.CoinPolicyModificationFeature, false)
	return nil
}
