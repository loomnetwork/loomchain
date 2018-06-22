package plasma_cash

import (
	"testing"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

/*
func TestPlasmaCashSMT(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	contract := &PlasmaCash{}
	err := contract.Init(ctx, &InitRequest{})

	require.Nil(t, err)

	//verify blockheight is zero
	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)

	assert.Equal(t, uint64(0), pbk.CurrentHeight.Value.Uint64(), "height should be 1")

	req := &PlasmaTxRequest{}
	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	//verify that blockheight is now one
	ctx.Get(blockHeightKey, pbk)
	assert.Equal(t, uint64(1), pbk.CurrentHeight.Value.Uint64(), "height should be 1")

	pb := &PlasmaBlock{}
	ctx.Get([]byte("pcash_block_1"), pb)

	require.NotNil(t, pb)

	reqBlockReq := &GetCurrentBlockRequest{}
	//Make sure we can also call the current blockheight transaction
	res, err := contract.GetCurrentBlockRequest(ctx, reqBlockReq)

	require.Nil(t, err)
	require.NotNil(t, res)
	assert.Equal(t, uint64(1), res.BlockHeight.Value.Uint64(), "height should be 1")
}
*/

//TODO change to bigint
func TestRound(t *testing.T) {
	res := round(9, int64(1000))
	assert.Equal(t, res, int64(1000))
	res = round(999, 1000)
	assert.Equal(t, res, int64(1000))
	res = round(0, 1000)
	assert.Equal(t, res, int64(1000))
	res = round(1000, 1000)
	assert.Equal(t, res, int64(2000))
	res = round(1001, 1000)
	assert.Equal(t, res, int64(2000))
	res = round(1999, 1000)
	assert.Equal(t, res, int64(2000))
}

func TestPlasmaCashSMT(t *testing.T) {
	fakeCtx := plugin.CreateFakeContext(addr1, addr1)
	ctx := contractpb.WrapPluginContext(
		fakeCtx,
	)

	contract := &PlasmaCash{}
	err := contract.Init(ctx, &InitRequest{})
	require.Nil(t, err)

	pending := &Pending{}
	ctx.Get(pendingTXsKey, pending)
	assert.Equal(t, len(pending.Transactions), 0, "length should be zero")

	plasmaTx := &PlasmaTx{
		Slot: 0,
	}
	req := &PlasmaTxRequest{
		Plasmatx: plasmaTx,
	}
	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	ctx.Get(pendingTXsKey, pending)
	assert.NotEqual(t, len(pending.Transactions), 0, "length should not be zero")

	reqMainnet := &SubmitBlockToMainnetRequest{}
	err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	require.NotNil(t, fakeCtx.Events[0])
	assert.Equal(t, fakeCtx.Events[0].Topics[0], "pcash_mainnet_merkle", "incorrect topic")
	//TODO	assert.Equal(t, fakeCtx.Events[0].Event, []byte("asdfb"), "incorrect merkle hash")

	//Ok lets get the same block back
	reqBlock := &GetBlockRequest{}
	reqBlock.BlockHeight = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(1000),
	}
	resblock, err := contract.GetBlockRequest(ctx, reqBlock)
	require.Nil(t, err)
	require.NotNil(t, resblock)

	reqMainnet2 := &SubmitBlockToMainnetRequest{}
	err = contract.SubmitBlockToMainnet(ctx, reqMainnet2)
	require.Nil(t, err)

	reqBlock2 := &GetBlockRequest{}
	reqBlock2.BlockHeight = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(2000),
	}
	resblock2, err := contract.GetBlockRequest(ctx, reqBlock2)
	require.Nil(t, err)
	require.NotNil(t, resblock2)
}
