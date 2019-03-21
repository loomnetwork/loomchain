package auth

import (
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
)

var (
	multiChainFaature = "mutichain"
)

func NewChainConfigMiddleware(
	chains map[string]ChainConfig,
	createAddressMapperCtx func(state loomchain.State) (contractpb.Context, error),
) loomchain.TxMiddlewareFunc {
	MultiChainSignatureTxMiddleware := NewMultiChainSignatureTxMiddleware(chains, createAddressMapperCtx)
	DefaultVal := (len(chains) > 0)
	return loomchain.TxMiddlewareFunc(func(
		state loomchain.State,
		txBytes []byte,
		next loomchain.TxHandlerFunc,
		isCheckTx bool,
	) (loomchain.TxHandlerResult, error) {
		if state.FeatureEnabled(multiChainFaature, DefaultVal) {
			return MultiChainSignatureTxMiddleware(state, txBytes, next, isCheckTx)
		} else {
			return SignatureTxMiddleware(state, txBytes, next, isCheckTx)
		}
	})
}
