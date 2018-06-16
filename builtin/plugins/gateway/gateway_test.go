package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/loomnetwork/go-loom"
	ctypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

var (
	addr1       = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2       = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	ethAccAddr1 = loom.MustParseAddress("eth:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestInit(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	gw := &Gateway{}
	err := gw.Init(ctx, &GatewayInitRequest{})
	assert.Nil(t, err)

	resp, err := gw.GetState(ctx, &GatewayStateRequest{})
	assert.Nil(t, err)
	s := resp.State
	assert.Equal(t, uint64(0), s.LastEthBlock)
}

func TestEmptyEventBatchProcessing(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	contract := &Gateway{}
	err := contract.Init(ctx, &GatewayInitRequest{})
	assert.Nil(t, err)

	// Should error out on an empty batch
	err = contract.ProcessEventBatchRequest(ctx, &ProcessEventBatchRequest{})
	assert.NotNil(t, err)
}

func TestOldEventBatchProcessing(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	contract := &Gateway{}
	err := contract.Init(ctx, &GatewayInitRequest{})
	assert.Nil(t, err)

	// Events from a specific block should only be processed once
	err = contract.ProcessEventBatchRequest(ctx, &ProcessEventBatchRequest{
		FtDeposits: genTokenDeposits([]uint64{5}),
	})
	assert.Nil(t, err)
	resp, err := contract.GetState(ctx, &GatewayStateRequest{})
	assert.Nil(t, err)
	s := resp.State
	assert.Equal(t, uint64(5), s.LastEthBlock)

	err = contract.ProcessEventBatchRequest(ctx, &ProcessEventBatchRequest{
		FtDeposits: genTokenDeposits([]uint64{5}),
	})
	assert.NotNil(t, err)
	resp, err = contract.GetState(ctx, &GatewayStateRequest{})
	assert.Nil(t, err)
	s = resp.State
	assert.Equal(t, uint64(5), s.LastEthBlock)
}

func TestOutOfOrderEventBatchProcessing(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	contract := &Gateway{}
	err := contract.Init(ctx, &GatewayInitRequest{})
	assert.Nil(t, err)

	// Batch must have events ordered by block (lowest to highest)
	err = contract.ProcessEventBatchRequest(ctx, &ProcessEventBatchRequest{
		FtDeposits: genTokenDeposits([]uint64{10, 9}),
	})
	assert.NotNil(t, err)
}

func TestEthDeposit(t *testing.T) {
	fakeCtx := plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/)
	gwCtx := contract.WrapPluginContext(fakeCtx)
	ethCoinAddr := fakeCtx.CreateContract(coin.Contract)
	coinCtx := contract.WrapPluginContext(fakeCtx.WithAddress(ethCoinAddr))

	ethCoin := &coin.Coin{}
	err := ethCoin.Init(coinCtx, &ctypes.InitRequest{})
	assert.Nil(t, err)

	ethToken := loom.RootAddress("eth")
	gw := &Gateway{}
	err = gw.Init(gwCtx, &GatewayInitRequest{
		Tokens: []*GatewayTokenMapping{&GatewayTokenMapping{
			FromToken: ethToken.MarshalPB(),
			ToToken:   ethCoinAddr.MarshalPB(),
		}},
	})
	assert.Nil(t, err)

	dappAcct1 := ethAccAddr1
	dappAcct1.ChainID = "chain"
	dappAcct1PB := dappAcct1.MarshalPB()
	balResp, err := ethCoin.BalanceOf(coinCtx, &ctypes.BalanceOfRequest{
		Owner: dappAcct1PB,
	})
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), balResp.Balance.Value.Uint64())

	depositAmount := int64(10)
	err = gw.ProcessEventBatchRequest(gwCtx, &ProcessEventBatchRequest{
		FtDeposits: []*TokenDeposit{
			&TokenDeposit{
				Token:    ethToken.MarshalPB(),
				From:     ethAccAddr1.MarshalPB(),
				To:       dappAcct1PB,
				Amount:   &types.BigUInt{Value: *loom.NewBigUIntFromInt(depositAmount)},
				EthBlock: 5,
			},
		},
	})
	assert.Nil(t, err)

	balResp, err = ethCoin.BalanceOf(coinCtx, &ctypes.BalanceOfRequest{
		Owner: dappAcct1PB,
	})
	assert.Nil(t, err)
	assert.Equal(t, depositAmount, balResp.Balance.Value.Int64())
}

func genTokenDeposits(blocks []uint64) []*TokenDeposit {
	ethToken := loom.RootAddress("eth")
	result := []*TokenDeposit{}
	for _, b := range blocks {
		for i := 0; i < 5; i++ {
			result = append(result, &TokenDeposit{
				Token: ethToken.MarshalPB(),
				From:  addr2.MarshalPB(),
				Amount: &types.BigUInt{
					Value: *loom.NewBigUIntFromInt(int64(i + 1)),
				},
				EthBlock: b,
			})
		}
	}
	return result
}
