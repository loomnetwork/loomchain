package auth

import (
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
)

const (
	chainFeaturePrefix = "auth:sigtx:"
	defaultChain       = "default"
)

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
		if len(authConfig.Chains) > 0 {
			chains := getEnabledChains(authConfig.Chains, state)
			multiChainSignatureTxMiddleware := NewMultiChainSignatureTxMiddleware(chains, createAddressMapperCtx)
			return multiChainSignatureTxMiddleware(state, txBytes, next, isCheckTx)
		}

		return SignatureTxMiddleware(state, txBytes, next, isCheckTx)
	})
}

func getEnabledChains(chains map[string]ChainConfig, state loomchain.State) map[string]ChainConfig {
	enabledChains := map[string]ChainConfig{}
	for name, config := range chains {
		if name == defaultChain {
			enabledChains[name] = config
			continue
		}
		if state.FeatureEnabled(chainFeaturePrefix+name, false) {
			enabledChains[name] = config
		}
	}
	return enabledChains
}
