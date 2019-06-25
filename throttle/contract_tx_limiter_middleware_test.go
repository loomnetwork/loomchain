package throttle

import (
	"context"
	"testing"

	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/loomchain"
	loomAuth "github.com/loomnetwork/loomchain/auth"
	udw "github.com/loomnetwork/loomchain/builtin/plugins/user_deployer_whitelist"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

func TestContractTxLimiterMiddleware(t *testing.T) {
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{Height: 5}, nil, nil)
	//EVMTxn
	txSignedEVM := mockSignedTx(t, uint64(2), callId, vm.VMType_EVM, contract)

	//init contract
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	dwAddr := fakeCtx.CreateContract(udw.Contract)
	contractContext := contractpb.WrapPluginContext(fakeCtx.WithAddress(dwAddr))
	udwContract := &udw.UserDeployerWhitelist{}
	tier := &udwtypes.TierInfo{
		TierID:     udwtypes.TierID_DEFAULT,
		Fee:        100,
		Name:       "Tier1",
		BlockRange: 10,
		MaxTx:      1,
	}
	tierList := []*udwtypes.TierInfo{}
	tierList = append(tierList, tier)
	err := udwContract.Init(contractContext, &udwtypes.InitRequest{
		Owner:    owner.MarshalPB(),
		TierInfo: tierList,
	})

	require.NoError(t, err)
	// create middleware
	cfg := DefaultContractTxLimiterConfig()
	contractTxLimiterMiddleware := NewContractTxLimiterMiddleware(cfg,
		func(state loomchain.State) (contractpb.Context, error) {
			return contractContext, nil
		},
	)

	ctx := context.WithValue(state.Context(), loomAuth.ContextKeyOrigin, owner)
	_, err = throttleMiddlewareHandlerCheckTx(contractTxLimiterMiddleware, state, txSignedEVM, ctx)
	require.NoError(t, err)
	_, err = throttleMiddlewareHandlerCheckTx(contractTxLimiterMiddleware, state, txSignedEVM, ctx)
	require.NoError(t, err)
	_, err = throttleMiddlewareHandlerCheckTx(contractTxLimiterMiddleware, state, txSignedEVM, ctx)
	require.NoError(t, err)

}
