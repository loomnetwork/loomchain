package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestInit(t *testing.T) {
	ctx := contract.WrapPluginContext(
		plugin.CreateFakeContext(addr1 /*caller*/, addr1 /*contract*/),
	)

	contract := &Gateway{}
	err := contract.Init(ctx, &GatewayInitRequest{})
	assert.Nil(t, err)

	resp, err := contract.GetState(ctx, &GatewayStateRequest{})
	assert.Nil(t, err)
	s := resp.State
	assert.Equal(t, uint64(0), s.LastEthBlock)
	assert.Equal(t, uint64(0), s.EthBalance.Value.Uint64())
	assert.Equal(t, uint64(0), s.Erc20Balance.Value.Uint64())
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
		FtDeposits: genFTDeposits([]uint64{5}),
	})
	assert.Nil(t, err)
	resp, err := contract.GetState(ctx, &GatewayStateRequest{})
	assert.Nil(t, err)
	s := resp.State
	assert.Equal(t, uint64(5), s.LastEthBlock)
	assert.Equal(t, uint64(15), s.EthBalance.Value.Uint64())
	assert.Equal(t, uint64(0), s.Erc20Balance.Value.Uint64())
	err = contract.ProcessEventBatchRequest(ctx, &ProcessEventBatchRequest{
		FtDeposits: genFTDeposits([]uint64{5}),
	})
	assert.NotNil(t, err)
	resp, err = contract.GetState(ctx, &GatewayStateRequest{})
	assert.Nil(t, err)
	s = resp.State
	assert.Equal(t, uint64(5), s.LastEthBlock)
	assert.Equal(t, uint64(15), s.EthBalance.Value.Uint64())
	assert.Equal(t, uint64(0), s.Erc20Balance.Value.Uint64())
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
		FtDeposits: genFTDeposits([]uint64{10, 9}),
	})
	assert.NotNil(t, err)
}

func genFTDeposits(blocks []uint64) []*FTDeposit {
	ethToken := loom.RootAddress("eth")
	result := []*FTDeposit{}
	for _, b := range blocks {
		for i := 0; i < 5; i++ {
			result = append(result, &FTDeposit{
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
