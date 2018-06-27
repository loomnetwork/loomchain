package plasma_cash

import (
	"fmt"
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

//TODO add test to verify idempodency

func TestRound(t *testing.T) {
	//TODO change to bigint
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
		Slot: 5,
	}
	req := &PlasmaTxRequest{
		Plasmatx: plasmaTx,
	}
	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	ctx.Get(pendingTXsKey, pending)
	assert.Equal(t, len(pending.Transactions), 1, "length should be one")

	reqMainnet := &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	require.NotNil(t, fakeCtx.Events[0])
	assert.Equal(t, fakeCtx.Events[0].Topics[0], "pcash_mainnet_merkle", "incorrect topic")
	assert.Equal(t, 32, len(fakeCtx.Events[0].Event), "incorrect merkle hash length")
	//	assert.Equal(t, fakeCtx.Events[0].Event, []byte("asdfb"), "incorrect merkle hash")

	//Ok lets get the same block back
	reqBlock := &GetBlockRequest{
		BlockHeight: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(1000),
		},
	}
	resblock, err := contract.GetBlockRequest(ctx, reqBlock)
	require.Nil(t, err)
	require.NotNil(t, resblock)

	assert.Equal(t, 1, len(resblock.Block.Transactions), "incorrect number of saved transactions")

	reqMainnet2 := &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet2)
	require.Nil(t, err)

	reqBlock2 := &GetBlockRequest{}
	reqBlock2.BlockHeight = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(2000),
	}
	resblock2, err := contract.GetBlockRequest(ctx, reqBlock2)
	require.Nil(t, err)
	require.NotNil(t, resblock2)
}

func TestSha3Encodings(t *testing.T) {

	res, err := soliditySha3(5)
	assert.Equal(t, fmt.Sprintf("0x%x", res), "0xfe07a98784cd1850eae35ede546d7028e6bf9569108995fc410868db775e5e6a", "incorrect sha3 hex")
	require.Nil(t, err)

	res2, err := soliditySha3(3)
	assert.Equal(t, fmt.Sprintf("0x%x", res2), "0xd4c69e49e83a6047f46e42b2d053a1f0c6e70ea42862e5ef4ad66b3666c5e2af", "incorrect sha3 hex")
	require.Nil(t, err)

}

func TestRLPEncodings(t *testing.T) {
	address, err := loom.LocalAddressFromHexString("0x5194b63f10691e46635b27925100cfc0a5ceca62")
	require.Nil(t, err)

	fmt.Printf("address-%v", address)
	plasmaTx := &PlasmaTx{
		Slot: 5,
		PreviousBlock: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(0),
		},
		Denomination: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(1),
		},
		NewOwner: &types.Address{ChainId: "default",
			Local: address},
	}

	data, err := rlpEncode(plasmaTx)
	assert.Equal(t, "d8058001945194b63f10691e46635b27925100cfc0a5ceca62", fmt.Sprintf("%x", data), "incorrect sha3 hex")
	require.Nil(t, err)

}
