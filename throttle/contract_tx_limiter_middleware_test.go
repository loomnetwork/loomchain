package throttle

import (
	"context"
	"testing"

	"github.com/loomnetwork/go-loom"
	dwtypes "github.com/loomnetwork/go-loom/builtin/types/deployer_whitelist"
	udwtypes "github.com/loomnetwork/go-loom/builtin/types/user_deployer_whitelist"
	goloomplugin "github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	loomAuth "github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	"github.com/loomnetwork/loomchain/builtin/plugins/deployer_whitelist"
	udw "github.com/loomnetwork/loomchain/builtin/plugins/user_deployer_whitelist"
	"github.com/loomnetwork/loomchain/store"
	"github.com/loomnetwork/loomchain/vm"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
)

var (
	// addr1        = loom.MustParseAddress("default:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	addr3        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c5")
	addr4        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c7")
	addr5        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c9")
	addr6        = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e0112")
	contractAddr = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01ab")
	user         = addr3.MarshalPB()
	chainID      = "default"
)

func TestContractTxLimiterMiddleware(t *testing.T) {
	//init contract
	fakeCtx := goloomplugin.CreateFakeContext(addr1, addr1)
	fakeCtx.SetFeature(loomchain.CoinVersion1_1Feature, true)
	udwAddr := fakeCtx.CreateContract(udw.Contract)
	udwContext := fakeCtx.WithAddress(udwAddr)
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
	err := udwContract.Init(contractpb.WrapPluginContext(udwContext), &udwtypes.InitRequest{
		Owner:    owner.MarshalPB(),
		TierInfo: tierList,
	})
	require.NoError(t, err)

	// Deploy deployerWhitelist and coin contract to deploy a contract for initializing contractToTierMap
	dwContract := &deployer_whitelist.DeployerWhitelist{}
	dwAddr := fakeCtx.CreateContract(deployer_whitelist.Contract)
	dctx := fakeCtx.WithAddress(dwAddr)
	err = dwContract.Init(contractpb.WrapPluginContext(dctx), &deployer_whitelist.InitRequest{
		Owner: addr4.MarshalPB(),
		Deployers: []*dwtypes.Deployer{
			&dwtypes.Deployer{
				Address: addr5.MarshalPB(),
				Flags:   uint32(1),
			},
		},
	})
	require.Nil(t, err)

	coinContract := &coin.Coin{}
	coinAddr := fakeCtx.CreateContract(coin.Contract)
	coinCtx := fakeCtx.WithAddress(coinAddr)
	err = coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			{Owner: user, Balance: uint64(300)},
		},
	})
	require.Nil(t, err)

	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(18), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(1000))
	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr3)), &coin.ApproveRequest{
		Spender: udwAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *ret},
	})
	require.Nil(t, err)

	err = udwContract.AddUserDeployer(contractpb.WrapPluginContext(udwContext.WithSender(addr3)),
		&udwtypes.WhitelistUserDeployerRequest{
			DeployerAddr: addr1.MarshalPB(),
			TierID:       0,
		})
	require.Nil(t, err)
	// record deployment
	err = udw.RecordEVMContractDeployment(contractpb.WrapPluginContext(udwContext.WithSender(addr3)),
		addr1, contractAddr)
	require.Nil(t, err)

	// create middleware
	state := loomchain.NewStoreState(nil, store.NewMemStore(), abci.Header{Height: 5}, nil, nil)
	//EVMTxn
	txSignedEVM := mockSignedTx(t, uint64(2), callId, vm.VMType_EVM, contractAddr)

	cfg := DefaultContractTxLimiterConfig()
	contractTxLimiterMiddleware := NewContractTxLimiterMiddleware(cfg,
		func(state loomchain.State) (contractpb.Context, error) {
			return contractpb.WrapPluginContext(udwContext), nil
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
