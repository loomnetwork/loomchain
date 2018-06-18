package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

var (
	addr1        = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2        = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	ethAccAddr1  = loom.MustParseAddress("eth:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	ethTokenAddr = loom.RootAddress("eth")
)

const (
	coinDecimals = 18
)

func TestInit(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	gw := &Gateway{}
	err := gw.Init(ctx, &GatewayInitRequest{})
	require.Nil(t, err)

	resp, err := gw.GetState(ctx, &GatewayStateRequest{})
	require.Nil(t, err)
	s := resp.State
	assert.Equal(t, uint64(0), s.LastEthBlock)
}

func TestEmptyEventBatchProcessing(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	contract := &Gateway{}
	err := contract.Init(ctx, &GatewayInitRequest{})
	require.Nil(t, err)

	// Should error out on an empty batch
	err = contract.ProcessEventBatchRequest(ctx, &ProcessEventBatchRequest{})
	require.NotNil(t, err)
}

func TestOldEventBatchProcessing(t *testing.T) {
	fakeCtx := plugin.CreateFakeContext(addr1 /*caller*/, loom.Address{} /*contract*/)
	gw := &Gateway{}
	gwAddr := fakeCtx.CreateContract(contract.MakePluginContract(gw))
	gwCtx := contract.WrapPluginContext(fakeCtx.WithAddress(gwAddr))

	coinContract, err := deployCoinContract(fakeCtx, gwAddr, 100000)
	require.Nil(t, err)
	initialGatewayCoinBal := sciNot(100000, coinDecimals)

	err = gw.Init(gwCtx, &GatewayInitRequest{
		Tokens: []*GatewayTokenMapping{&GatewayTokenMapping{
			FromToken: ethTokenAddr.MarshalPB(),
			ToToken:   coinContract.Address.MarshalPB(),
		}},
	})
	require.Nil(t, err)

	coinBal, err := coinContract.getBalance(fakeCtx, gwAddr)
	require.Nil(t, err)
	assert.Equal(t, initialGatewayCoinBal, coinBal, "gateway account balance should match initial balance")

	err = gw.ProcessEventBatchRequest(gwCtx, &ProcessEventBatchRequest{
		FtDeposits: genTokenDeposits([]uint64{5}),
	})
	require.Nil(t, err)
	resp, err := gw.GetState(gwCtx, &GatewayStateRequest{})
	require.Nil(t, err)
	s := resp.State
	assert.Equal(t, uint64(5), s.LastEthBlock)

	coinBal, err = coinContract.getBalance(fakeCtx, gwAddr)
	require.Nil(t, err)
	assert.True(t, coinBal.Cmp(initialGatewayCoinBal) < 0, "gateway account balance should have been reduced")

	// Events from each block should only be processed once, even if multiple batches contain the
	// same block.
	err = gw.ProcessEventBatchRequest(gwCtx, &ProcessEventBatchRequest{
		FtDeposits: genTokenDeposits([]uint64{5}),
	})
	require.NotNil(t, err)
	resp, err = gw.GetState(gwCtx, &GatewayStateRequest{})
	require.Nil(t, err)
	s = resp.State
	assert.Equal(t, uint64(5), s.LastEthBlock)

	coinBal2, err := coinContract.getBalance(fakeCtx, gwAddr)
	require.Nil(t, err)
	assert.True(t, coinBal.Cmp(coinBal2) == 0, "gateway account balance should not have changed")
}

func TestOutOfOrderEventBatchProcessing(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	contract := &Gateway{}
	err := contract.Init(ctx, &GatewayInitRequest{})
	require.Nil(t, err)

	// Batch must have events ordered by block (lowest to highest)
	err = contract.ProcessEventBatchRequest(ctx, &ProcessEventBatchRequest{
		FtDeposits: genTokenDeposits([]uint64{10, 9}),
	})
	require.NotNil(t, err)
}

func TestEthDeposit(t *testing.T) {
	fakeCtx := plugin.CreateFakeContext(addr1 /*caller*/, loom.Address{} /*contract*/)
	gw := &Gateway{}
	gwAddr := fakeCtx.CreateContract(contract.MakePluginContract(gw))
	gwCtx := contract.WrapPluginContext(fakeCtx.WithAddress(gwAddr))

	coinContract, err := deployCoinContract(fakeCtx, gwAddr, 100000)
	err = gw.Init(gwCtx, &GatewayInitRequest{
		Tokens: []*GatewayTokenMapping{&GatewayTokenMapping{
			FromToken: ethTokenAddr.MarshalPB(),
			ToToken:   coinContract.Address.MarshalPB(),
		}},
	})
	require.Nil(t, err)

	dappAcct := ethAccAddr1
	dappAcct.ChainID = "chain"
	bal, err := coinContract.getBalance(fakeCtx, dappAcct)
	require.Nil(t, err)
	assert.Equal(t, uint64(0), bal.Uint64(), "receiver account balance should be zero")

	depositAmount := int64(10)
	err = gw.ProcessEventBatchRequest(gwCtx, &ProcessEventBatchRequest{
		FtDeposits: []*TokenDeposit{
			&TokenDeposit{
				Token:    ethTokenAddr.MarshalPB(),
				From:     ethAccAddr1.MarshalPB(),
				To:       dappAcct.MarshalPB(),
				Amount:   &types.BigUInt{Value: *loom.NewBigUIntFromInt(depositAmount)},
				EthBlock: 5,
			},
		},
	})
	require.Nil(t, err)

	bal2, err := coinContract.getBalance(fakeCtx, dappAcct)
	require.Nil(t, err)
	assert.Equal(t, depositAmount, bal2.Int64(), "receiver account balance should match deposit amount")
}

func genTokenDeposits(blocks []uint64) []*TokenDeposit {
	result := []*TokenDeposit{}
	for _, b := range blocks {
		for i := 0; i < 5; i++ {
			result = append(result, &TokenDeposit{
				Token: ethTokenAddr.MarshalPB(),
				From:  ethAccAddr1.MarshalPB(),
				To:    addr2.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *loom.NewBigUIntFromInt(int64(i + 1)),
				},
				EthBlock: b,
			})
		}
	}
	return result
}

type testCoinContract struct {
	Contract *coin.Coin
	Address  loom.Address
}

func deployCoinContract(ctx *plugin.FakeContext, gwAddr loom.Address, bal uint64) (*testCoinContract, error) {
	// Deploy the coin contract & give the gateway contract a bunch of coins
	coinContract := &coin.Coin{}
	coinAddr := ctx.CreateContract(contract.MakePluginContract(coinContract))
	coinCtx := contract.WrapPluginContext(ctx.WithAddress(coinAddr))

	err := coinContract.Init(coinCtx, &ctypes.InitRequest{
		Accounts: []*coin.InitialAccount{
			&coin.InitialAccount{
				Owner:   gwAddr.MarshalPB(),
				Balance: bal,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return &testCoinContract{
		Contract: coinContract,
		Address:  coinAddr,
	}, nil
}

func (c *testCoinContract) getBalance(ctx *plugin.FakeContext, ownerAddr loom.Address) (*loom.BigUInt, error) {
	resp, err := c.Contract.BalanceOf(
		contract.WrapPluginContext(ctx.WithAddress(c.Address)),
		&coin.BalanceOfRequest{Owner: ownerAddr.MarshalPB()},
	)
	if err != nil {
		return loom.NewBigUIntFromInt(0), err
	}
	return &resp.Balance.Value, nil
}

func sciNot(m, n int64) *loom.BigUInt {
	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(n), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(m))
	return ret
}
