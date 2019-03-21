package auth

import (
	"fmt"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
)

const (
	chainFeaturePrefix = "auth:sigtx:"
)

// NewChainConfigMiddleware returns middleware that verifies signed txs using either
// SignedTxMiddleware or MultiChainSignatureTxMiddleware, it switches the underlying middleware
// based on the on-chain and off-chain auth config settings.
func NewChainConfigMiddleware(
	authConfig *Config,
	createAddressMapperCtx func(state loomchain.State) (contractpb.Context, error),
) loomchain.TxMiddlewareFunc {
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (loomchain.TxHandlerResult, error) {
		chains := getEnabledChains(authConfig.Chains, state)
		if len(chains) > 0 {
			mw := NewMultiChainSignatureTxMiddleware(chains, createAddressMapperCtx)
			return mw(state, txBytes, next, isCheckTx)
		}

		return SignatureTxMiddleware(state, txBytes, next, isCheckTx)
	})
}

func getEnabledChains(chains map[string]ChainConfig, state loomchain.State) map[string]ChainConfig {
	enabledChains := map[string]ChainConfig{}
	for chainID, config := range chains {
		if chainID == state.Block().ChainID {
			enabledChains[chainID] = config
			continue
		}
		if state.FeatureEnabled(chainFeaturePrefix+chainID, false) {
			enabledChains[chainID] = config
		}
	}
	return enabledChains
}

// ResolveAccountAddress takes a local or foreign account address and returns the address used
// to identify the account on this chain.
func ResolveAccountAddress(
	account loom.Address, state loomchain.State, authCfg *Config,
	createAddressMapperCtx func(state loomchain.State) (contractpb.Context, error),
) (loom.Address, error) {
	chains := getEnabledChains(authCfg.Chains, state)
	if len(chains) > 0 {
		chain, found := authCfg.Chains[account.ChainID]
		if !found {
			return loom.Address{}, fmt.Errorf("unknown chain ID %s", account.ChainID)
		}

		switch chain.AccountType {
		case NativeAccountType:
			return account, nil

		case MappedAccountType:
			addr, err := getMappedAccountAddress(state, account, createAddressMapperCtx)
			if err != nil {
				return loom.Address{}, err
			}
			return addr, nil

		default:
			return loom.Address{},
				fmt.Errorf("invalid account type %v for chain ID %s", chain.AccountType, account.ChainID)
		}
	}

	if account.ChainID != state.Block().ChainID {
		return loom.Address{}, fmt.Errorf("unknown chain ID %s", account.ChainID)
	}
	return account, nil
}
