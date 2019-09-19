package auth

import (
	"fmt"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/features"
	"github.com/loomnetwork/loomchain/state"
)

// NewChainConfigMiddleware returns middleware that verifies signed txs using either
// SignedTxMiddleware or MultiChainSignatureTxMiddleware, it switches the underlying middleware
// based on the on-chain and off-chain auth config settings.
func NewChainConfigMiddleware(
	authConfig *Config,
	createAddressMapperCtx func(s state.State) (contractpb.StaticContext, error),
) loomchain.TxMiddlewareFunc {
	return loomchain.TxMiddlewareFunc(func(
		s state.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (loomchain.TxHandlerResult, error) {
		chains := getEnabledChains(authConfig.Chains, s)
		if len(chains) > 0 {
			mw := NewMultiChainSignatureTxMiddleware(chains, createAddressMapperCtx)
			return mw(s, txBytes, next, isCheckTx)
		}

		return SignatureTxMiddleware(s, txBytes, next, isCheckTx)
	})
}

// Filters out any auth.ChainConfig(s) that haven't been enabled by the majority of validators.
func getEnabledChains(chains map[string]ChainConfig, s state.State) map[string]ChainConfig {
	enabledChains := map[string]ChainConfig{}
	for chainID, config := range chains {
		if s.FeatureEnabled(features.AuthSigTxFeaturePrefix+chainID, false) {
			enabledChains[chainID] = config
		}
	}

	// Ensure the chain will always process native ed25519 txs by default.
	if len(enabledChains) > 0 {
		curChainID := s.Block().ChainID
		if _, found := enabledChains[curChainID]; !found {
			enabledChains[curChainID] = ChainConfig{
				TxType:      LoomSignedTxType,
				AccountType: NativeAccountType,
			}
		}
	}
	return enabledChains
}

// ResolveAccountAddress takes a local or foreign account address and returns the address used
// to identify the account on this chain.
func ResolveAccountAddress(
	account loom.Address, s state.State, authCfg *Config,
	createAddressMapperCtx func(s state.State) (contractpb.StaticContext, error),
) (loom.Address, error) {
	chains := getEnabledChains(authCfg.Chains, s)
	if len(chains) > 0 {
		chain, found := chains[account.ChainID]
		if !found {
			return loom.Address{}, fmt.Errorf("unknown chain ID %s", account.ChainID)
		}

		switch chain.AccountType {
		case NativeAccountType:
			return account, nil

		case MappedAccountType:
			addr, err := getMappedAccountAddress(s, account, createAddressMapperCtx)
			if err != nil {
				return loom.Address{}, err
			}
			return addr, nil

		default:
			return loom.Address{},
				fmt.Errorf("invalid account type %v for chain ID %s", chain.AccountType, account.ChainID)
		}
	}

	if account.ChainID != s.Block().ChainID {
		return loom.Address{}, fmt.Errorf("unknown chain ID %s", account.ChainID)
	}
	return account, nil
}
