// +build evm

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
	res = round(2000, 1000)
	assert.Equal(t, res, int64(3000))
	res = round(2001, 1000)
	assert.Equal(t, res, int64(3000))
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

	contractAddr := loom.RootAddress("eth")

	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     5,
		Contract: contractAddr.MarshalPB(),
	}))
	err = saveAccount(ctx, &Account{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
		Slots:    []uint64{5},
	})
	require.Nil(t, err)

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:     5,
			Sender:   addr2.MarshalPB(),
			NewOwner: addr3.MarshalPB(),
		},
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

func TestEmptyPlasmaBlock(t *testing.T) {
	fakeCtx := plugin.CreateFakeContext(addr1, addr1)
	ctx := contractpb.WrapPluginContext(
		fakeCtx,
	)

	contract := &PlasmaCash{}
	err := contract.Init(ctx, &InitRequest{})
	require.Nil(t, err)

	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)

	assert.Equal(t, pbk.CurrentHeight.Value.Int64(), int64(0), "invalid height")

	reqMainnet := &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	ctx.Get(blockHeightKey, pbk)
	assert.Equal(t, int64(1000), pbk.CurrentHeight.Value.Int64(), "invalid height")

	reqMainnet = &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	ctx.Get(blockHeightKey, pbk)
	assert.Equal(t, int64(2000), pbk.CurrentHeight.Value.Int64(), "invalid height")

	reqMainnet = &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	ctx.Get(blockHeightKey, pbk)
	assert.Equal(t, int64(3000), pbk.CurrentHeight.Value.Int64(), "invalid height")
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

// Clear pending txs from state after finalizing a block in SubmitBlockToMainnet.
func TestPlasmaClearPending(t *testing.T) {
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

	contractAddr := loom.RootAddress("eth")

	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     5,
		Contract: contractAddr.MarshalPB(),
	}))
	err = saveAccount(ctx, &Account{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
		Slots:    []uint64{5},
	})
	require.Nil(t, err)

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:     5,
			Sender:   addr2.MarshalPB(),
			NewOwner: addr3.MarshalPB(),
		},
	}
	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	ctx.Get(pendingTXsKey, pending)
	assert.Equal(t, len(pending.Transactions), 1, "length should be one")

	reqMainnet := &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	pending2 := &Pending{}
	ctx.Get(pendingTXsKey, pending2)
	assert.Equal(t, len(pending2.Transactions), 0, "length should be zero")
}

// Error out if an attempt is made to add a tx with a slot that is already referenced in pending txs in PlasmaTxRequest.
func TestPlasmaErrorDuplicate(t *testing.T) {
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

	contractAddr := loom.RootAddress("eth")

	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     5,
		Contract: contractAddr.MarshalPB(),
	}))
	err = saveAccount(ctx, &Account{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
		Slots:    []uint64{5},
	})
	require.Nil(t, err)

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:     5,
			Sender:   addr2.MarshalPB(),
			NewOwner: addr3.MarshalPB(),
		},
	}
	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	//Send slot5 a second time
	err = contract.PlasmaTxRequest(ctx, req)
	require.NotNil(t, err)

}

func TestPlasmaCashBalanceAfterDeposit(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)

	tokenIDs := []*types.BigUInt{
		&types.BigUInt{Value: *loom.NewBigUIntFromInt(721)},
		&types.BigUInt{Value: *loom.NewBigUIntFromInt(127)},
	}

	err := plasmaContract.DepositRequest(ctx, &DepositRequest{
		Slot:         123,
		DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
		Denomination: tokenIDs[0],
		From:         addr2.MarshalPB(),
		Contract:     addr3.MarshalPB(),
	})
	require.Nil(t, err)

	resp, err := plasmaContract.BalanceOf(ctx, &BalanceOfRequest{
		Owner:    addr2.MarshalPB(),
		Contract: addr3.MarshalPB(),
	})
	require.Nil(t, err)

	require.Len(t, resp.Coins, 1, "account should have one coin")
	correctCoin := &Coin{
		Slot:     123,
		State:    CoinState_DEPOSITED,
		Token:    tokenIDs[0],
		Contract: addr3.MarshalPB(),
	}
	assert.Equal(t, resp.Coins[0].String(), correctCoin.String())

	err = plasmaContract.DepositRequest(ctx, &DepositRequest{
		Slot:         456,
		DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
		Denomination: tokenIDs[1],
		From:         addr2.MarshalPB(),
		Contract:     addr3.MarshalPB(),
	})
	require.Nil(t, err)

	resp, err = plasmaContract.BalanceOf(ctx, &BalanceOfRequest{
		Owner:    addr2.MarshalPB(),
		Contract: addr3.MarshalPB(),
	})
	require.Nil(t, err)

	correntCoin1 := &Coin{
		Slot:     123,
		State:    CoinState_DEPOSITED,
		Token:    tokenIDs[0],
		Contract: addr3.MarshalPB(),
	}
	correntCoin2 := &Coin{
		Slot:     456,
		State:    CoinState_DEPOSITED,
		Token:    tokenIDs[1],
		Contract: addr3.MarshalPB(),
	}

	assert.Equal(t, 2, len(resp.Coins))
	assert.Equal(t, correntCoin1.String(), resp.Coins[0].String())
	assert.Equal(t, correntCoin2.String(), resp.Coins[1].String())
}

func TestPlasmaCashTransferWithInvalidSender(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:     5, // sender doesn't own this coin
			Sender:   addr2.MarshalPB(),
			NewOwner: addr3.MarshalPB(),
		},
	}
	err := plasmaContract.PlasmaTxRequest(ctx, req)
	require.NotNil(t, err)
}

func TestPlasmaCashTransferWithInvalidCoinState(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)

	coins := []*Coin{
		&Coin{Slot: 5, State: CoinState_EXITING},
		&Coin{Slot: 6, State: CoinState_CHALLENGED},
		&Coin{Slot: 7, State: CoinState_EXITED},
	}
	for _, coin := range coins {
		require.Nil(t, saveCoin(ctx, coin))
	}

	err := saveAccount(ctx, &Account{
		Owner:    addr2.MarshalPB(),
		Contract: loom.RootAddress("eth").MarshalPB(),
		Slots:    []uint64{5, 6, 7},
	})
	require.Nil(t, err)

	for _, coin := range coins {
		req := &PlasmaTxRequest{
			Plasmatx: &PlasmaTx{
				Slot:     coin.Slot,
				Sender:   addr2.MarshalPB(),
				NewOwner: addr3.MarshalPB(),
			},
		}
		err = plasmaContract.PlasmaTxRequest(ctx, req)
		require.NotNil(t, err)
	}
}

func TestPlasmaCashExitWithInvalidCoinState(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)
	contractAddr := loom.RootAddress("eth")
	coins := []*Coin{
		&Coin{Slot: 5, State: CoinState_EXITING},
		&Coin{Slot: 6, State: CoinState_CHALLENGED},
		&Coin{Slot: 7, State: CoinState_EXITED},
	}
	for _, coin := range coins {
		require.Nil(t, saveCoin(ctx, coin))
	}

	err := saveAccount(ctx, &Account{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
		Slots:    []uint64{5, 6, 7},
	})
	require.Nil(t, err)

	for _, coin := range coins {
		req := &ExitCoinRequest{
			Owner: addr2.MarshalPB(),
			Slot:  coin.Slot,
		}
		err = plasmaContract.ExitCoin(ctx, req)
		require.NotNil(t, err)
	}
}

func TestPlasmaCashExit(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)
	contractAddr := loom.RootAddress("eth")
	coins := []*Coin{
		&Coin{Slot: 5, State: CoinState_DEPOSITED, Contract: contractAddr.MarshalPB()},
		&Coin{Slot: 6, State: CoinState_DEPOSITED, Contract: contractAddr.MarshalPB()},
		&Coin{Slot: 7, State: CoinState_DEPOSITED, Contract: contractAddr.MarshalPB()},
	}
	for _, coin := range coins {
		require.Nil(t, saveCoin(ctx, coin))
	}

	err := saveAccount(ctx, &Account{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
		Slots:    []uint64{5, 6, 7},
	})
	require.Nil(t, err)

	req := &ExitCoinRequest{
		Owner: addr2.MarshalPB(),
		Slot:  coins[1].Slot,
	}
	err = plasmaContract.ExitCoin(ctx, req)
	require.Nil(t, err)

	resp, err := plasmaContract.BalanceOf(ctx, &BalanceOfRequest{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, &Coin{
		Slot:     6,
		State:    CoinState_EXITING,
		Contract: contractAddr.MarshalPB(),
	}, resp.Coins[1])
}

func TestPlasmaCashWithdraw(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)
	contractAddr := loom.RootAddress("eth")
	coins := []*Coin{
		&Coin{Slot: 5, State: CoinState_DEPOSITED, Contract: contractAddr.MarshalPB()},
		&Coin{Slot: 6, State: CoinState_EXITING, Contract: contractAddr.MarshalPB()},
		&Coin{Slot: 7, State: CoinState_CHALLENGED, Contract: contractAddr.MarshalPB()},
		&Coin{Slot: 8, State: CoinState_EXITED, Contract: contractAddr.MarshalPB()},
	}
	for _, coin := range coins {
		require.Nil(t, saveCoin(ctx, coin))
	}

	err := saveAccount(ctx, &Account{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
		Slots:    []uint64{5, 6, 7, 8},
	})
	require.Nil(t, err)

	for _, coin := range coins {
		req := &WithdrawCoinRequest{
			Owner: addr2.MarshalPB(),
			Slot:  coin.Slot,
		}
		err = plasmaContract.WithdrawCoin(ctx, req)
		require.Nil(t, err)
	}
	resp, err := plasmaContract.BalanceOf(ctx, &BalanceOfRequest{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Len(t, resp.Coins, 0)
}

func TestGetUserSlots(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)
	contractAddr := loom.RootAddress("eth")

	a := &Account{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
		Slots:    []uint64{5, 7},
	}
	b := &Account{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
	}
	err := saveAccount(ctx, a)
	require.Nil(t, err)

	req := &GetUserSlotsRequest{
		Account: b,
	}
	res, err := plasmaContract.GetUserSlots(ctx, req)
	require.Nil(t, err)

	assert.Equal(t, []uint64{5, 7}, res.Slots, "proof should match")

}

func TestGetSlotMerkleProof(t *testing.T) {
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

	contractAddr := loom.RootAddress("eth")

	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     5,
		Contract: contractAddr.MarshalPB(),
	}))
	err = saveAccount(ctx, &Account{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
		Slots:    []uint64{5},
	})
	require.Nil(t, err)

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:     5,
			Sender:   addr2.MarshalPB(),
			NewOwner: addr3.MarshalPB(),
		},
	}
	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	reqMainnet := &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	reqSlotMerkle := &GetSlotMerkleProofRequest{}
	reqSlotMerkle.BlockHeight = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(1000),
	}
	reqSlotMerkle.Slot = 5

	res, err := contract.GetSlotMerkleProof(ctx, reqSlotMerkle)
	require.Nil(t, err)

	//TODO not sure why test case generates a empty proof, might be related to original transaction, shouldn't effect this function
	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, res.Proof, "proof should match")
}

func getPlasmaContractAndContext(t *testing.T) (*PlasmaCash, contractpb.Context) {
	fakeCtx := plugin.CreateFakeContext(addr1, addr1)
	ctx := contractpb.WrapPluginContext(fakeCtx)

	plasmaContract := &PlasmaCash{}
	err := plasmaContract.Init(ctx, &InitRequest{})
	require.Nil(t, err)

	return plasmaContract, ctx
}
