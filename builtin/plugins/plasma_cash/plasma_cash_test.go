// +build evm

package plasma_cash

import (
	"crypto/ecdsa"
	"fmt"
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/client/plasma_cash"
	"github.com/loomnetwork/go-loom/common/evmcompat"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ssha "github.com/miguelmota/go-solidity-sha3"

	"github.com/ethereum/go-ethereum/crypto"
	amtypes "github.com/loomnetwork/go-loom/builtin/types/address_mapper"
	pctypes "github.com/loomnetwork/go-loom/builtin/types/plasma_cash"
	"github.com/loomnetwork/loomchain/builtin/plugins/address_mapper"
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
	addressMapperAddress := fakeCtx.CreateContract(address_mapper.Contract)
	fakeCtx.RegisterContract("addressmapper", addressMapperAddress, addressMapperAddress)
	ctx := contractpb.WrapPluginContext(
		fakeCtx,
	)

	contract := &PlasmaCash{}
	err := contract.Init(ctx, &InitRequest{
		Oracle: addr1.MarshalPB(),
	})
	require.Nil(t, err)

	pending := &PendingTxs{}
	ctx.Get(pendingTXsKey, pending)
	assert.Equal(t, len(pending.Transactions), 0, "length should be zero")

	contractAddr := loom.RootAddress("eth")

	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     5,
		Contract: contractAddr.MarshalPB(),
	}))

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:          5,
			NewOwner:      addr3.MarshalPB(),
			PreviousBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			Denomination:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
		},
	}

	generatedSender, err := setupPlasmaTxAuth(ctx, req.Plasmatx, addr1)
	require.Nil(t, err)
	err = saveAccount(ctx, &Account{
		Owner: generatedSender.MarshalPB(),
		Slots: []uint64{5},
	})
	require.Nil(t, err)

	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	ctx.Get(pendingTXsKey, pending)
	assert.Equal(t, len(pending.Transactions), 1, "length should be one")

	reqMainnet := &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	require.NotNil(t, fakeCtx.Events[0])
	assert.Equal(t, fakeCtx.Events[0].Topics[0], "event:PlasmaCashTransferConfirmed", "incorrect topic")

	transferConfirmed := TransferConfirmed{}
	err = proto.Unmarshal(fakeCtx.Events[0].Event, &transferConfirmed)
	require.Nil(t, err)
	assert.Equal(t, loom.UnmarshalAddressPB(transferConfirmed.From).String(), generatedSender.String())
	assert.Equal(t, loom.UnmarshalAddressPB(transferConfirmed.To).String(), addr3.String())
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
		Value: *loom.NewBigUIntFromInt(1000),
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
	err := contract.Init(ctx, &InitRequest{
		Oracle: addr1.MarshalPB(),
	})
	require.Nil(t, err)

	pbk := &PlasmaBookKeeping{}
	ctx.Get(blockHeightKey, pbk)

	assert.Equal(t, pbk.CurrentHeight.Value.Int64(), int64(0), "invalid height")

	reqMainnet := &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	ctx.Get(blockHeightKey, pbk)
	assert.Equal(t, int64(0), pbk.CurrentHeight.Value.Int64(), "invalid height")

	reqMainnet = &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	ctx.Get(blockHeightKey, pbk)
	assert.Equal(t, int64(0), pbk.CurrentHeight.Value.Int64(), "invalid height")

	reqMainnet = &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	ctx.Get(blockHeightKey, pbk)
	assert.Equal(t, int64(0), pbk.CurrentHeight.Value.Int64(), "invalid height")
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
	addressMapperAddress := fakeCtx.CreateContract(address_mapper.Contract)
	fakeCtx.RegisterContract("addressmapper", addressMapperAddress, addressMapperAddress)
	ctx := contractpb.WrapPluginContext(
		fakeCtx,
	)

	contract := &PlasmaCash{}
	err := contract.Init(ctx, &InitRequest{
		Oracle: addr1.MarshalPB(),
	})
	require.Nil(t, err)

	pending := &PendingTxs{}
	ctx.Get(pendingTXsKey, pending)
	assert.Equal(t, len(pending.Transactions), 0, "length should be zero")

	contractAddr := loom.RootAddress("eth")

	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     5,
		Contract: contractAddr.MarshalPB(),
	}))

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:          5,
			NewOwner:      addr3.MarshalPB(),
			PreviousBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			Denomination:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
		},
	}

	generatedSender, err := setupPlasmaTxAuth(ctx, req.Plasmatx, addr1)
	require.Nil(t, err)
	err = saveAccount(ctx, &Account{
		Owner: generatedSender.MarshalPB(),
		Slots: []uint64{5},
	})
	require.Nil(t, err)

	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	ctx.Get(pendingTXsKey, pending)
	assert.Equal(t, len(pending.Transactions), 1, "length should be one")

	reqMainnet := &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	pending2 := &PendingTxs{}
	ctx.Get(pendingTXsKey, pending2)
	assert.Equal(t, len(pending2.Transactions), 0, "length should be zero")
}

// Error out if an attempt is made to add a tx with a slot that is already referenced in pending txs in PlasmaTxRequest.
func TestPlasmaErrorDuplicate(t *testing.T) {
	fakeCtx := plugin.CreateFakeContext(addr1, addr1)
	addressMapperAddress := fakeCtx.CreateContract(address_mapper.Contract)
	fakeCtx.RegisterContract("addressmapper", addressMapperAddress, addressMapperAddress)
	ctx := contractpb.WrapPluginContext(
		fakeCtx,
	)

	contract := &PlasmaCash{}
	err := contract.Init(ctx, &InitRequest{
		Oracle: addr1.MarshalPB(),
	})
	require.Nil(t, err)

	pending := &PendingTxs{}
	ctx.Get(pendingTXsKey, pending)
	assert.Equal(t, len(pending.Transactions), 0, "length should be zero")

	contractAddr := loom.RootAddress("eth")

	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     5,
		Contract: contractAddr.MarshalPB(),
	}))

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:          5,
			NewOwner:      addr3.MarshalPB(),
			PreviousBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			Denomination:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
		},
	}

	generatedSender, err := setupPlasmaTxAuth(ctx, req.Plasmatx, addr1)
	require.Nil(t, err)
	err = saveAccount(ctx, &Account{
		Owner: generatedSender.MarshalPB(),
		Slots: []uint64{5},
	})
	require.Nil(t, err)

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

	err := plasmaContract.ProcessRequestBatch(ctx, &pctypes.PlasmaCashRequestBatch{
		Requests: []*pctypes.PlasmaCashRequest{
			&pctypes.PlasmaCashRequest{
				Data: &pctypes.PlasmaCashRequest_Deposit{&pctypes.DepositRequest{
					Slot:         123,
					DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
					Denomination: tokenIDs[0],
					From:         addr2.MarshalPB(),
					Contract:     addr3.MarshalPB(),
				}},
				Meta: &pctypes.PlasmaCashEventMeta{
					BlockNumber: 1,
					LogIndex:    0,
					TxIndex:     0,
				},
			},
		},
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

	err = plasmaContract.ProcessRequestBatch(ctx, &pctypes.PlasmaCashRequestBatch{
		Requests: []*pctypes.PlasmaCashRequest{
			&pctypes.PlasmaCashRequest{
				Data: &pctypes.PlasmaCashRequest_Deposit{&pctypes.DepositRequest{
					Slot:         456,
					DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
					Denomination: tokenIDs[1],
					From:         addr2.MarshalPB(),
					Contract:     addr3.MarshalPB(),
				}},
				Meta: &pctypes.PlasmaCashEventMeta{
					BlockNumber: 2,
					LogIndex:    0,
					TxIndex:     0,
				},
			},
		},
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
			Slot:          5, // sender doesn't own this coin
			NewOwner:      addr3.MarshalPB(),
			PreviousBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			Denomination:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
		},
	}
	_, err := setupPlasmaTxAuth(ctx, req.Plasmatx, addr1)
	require.Nil(t, err)

	err = plasmaContract.PlasmaTxRequest(ctx, req)
	require.NotNil(t, err)
}

func TestPlasmaCashTxAuth(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)

	contractAddr := loom.RootAddress("eth")
	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     5,
		Contract: contractAddr.MarshalPB(),
	}))

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:          5,
			NewOwner:      addr3.MarshalPB(),
			PreviousBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			Denomination:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
		},
	}

	// Map addr2 against ethAddress instead of addr1
	generatedSender, err := setupPlasmaTxAuth(ctx, req.Plasmatx, addr2)
	require.Nil(t, err)

	err = saveAccount(ctx, &Account{
		Owner: generatedSender.MarshalPB(),
		Slots: []uint64{5},
	})
	require.Nil(t, err)

	// request wont go through as mapping wont be found
	err = plasmaContract.PlasmaTxRequest(ctx, req)
	require.Equal(t, err, ErrNotAuthorized)
}

func TestPlasmaCashTransferWithInvalidCoinState(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)

	ethPrivKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	coins := []*Coin{
		&Coin{Slot: 5, State: CoinState_EXITING},
		&Coin{Slot: 6, State: CoinState_CHALLENGED},
		&Coin{Slot: 7, State: CoinState_EXITED},
	}
	for _, coin := range coins {
		require.Nil(t, saveCoin(ctx, coin))
	}

	err = saveAccount(ctx, &Account{
		Owner: addr2.MarshalPB(),
		Slots: []uint64{5, 6, 7},
	})
	require.Nil(t, err)

	for _, coin := range coins {
		req := &PlasmaTxRequest{
			Plasmatx: &PlasmaTx{
				Slot:          coin.Slot,
				NewOwner:      addr3.MarshalPB(),
				PreviousBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
				Denomination:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			},
		}
		_, err := setupPlasmaTxAuthWithKey(ctx, ethPrivKey, req.Plasmatx, addr1)
		require.Nil(t, err)

		err = plasmaContract.PlasmaTxRequest(ctx, req)
		require.NotNil(t, err)
	}
}

func TestPlasmaCashExitWithInvalidCoinState(t *testing.T) {
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
		Owner: addr2.MarshalPB(),
		Slots: []uint64{5, 6, 7},
	})
	require.Nil(t, err)

	for i, coin := range coins {
		err = plasmaContract.ProcessRequestBatch(ctx, &pctypes.PlasmaCashRequestBatch{
			Requests: []*pctypes.PlasmaCashRequest{
				&pctypes.PlasmaCashRequest{
					Data: &pctypes.PlasmaCashRequest_StartedExit{&pctypes.PlasmaCashExitCoinRequest{
						Owner: addr2.MarshalPB(),
						Slot:  coin.Slot,
					}},
					Meta: &pctypes.PlasmaCashEventMeta{
						BlockNumber: uint64(i) + 1,
						LogIndex:    0,
						TxIndex:     0,
					},
				},
			},
		})
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
		Owner: addr2.MarshalPB(),
		Slots: []uint64{5, 6, 7},
	})
	require.Nil(t, err)

	err = plasmaContract.ProcessRequestBatch(ctx, &pctypes.PlasmaCashRequestBatch{
		Requests: []*pctypes.PlasmaCashRequest{
			&pctypes.PlasmaCashRequest{
				Data: &pctypes.PlasmaCashRequest_StartedExit{&pctypes.PlasmaCashExitCoinRequest{
					Owner: addr2.MarshalPB(),
					Slot:  coins[1].Slot,
				}},
				Meta: &pctypes.PlasmaCashEventMeta{
					BlockNumber: 1,
					LogIndex:    0,
					TxIndex:     0,
				},
			},
		},
	})
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
		Owner: addr2.MarshalPB(),
		Slots: []uint64{5, 6, 7, 8},
	})
	require.Nil(t, err)

	for i, coin := range coins {
		err = plasmaContract.ProcessRequestBatch(ctx, &pctypes.PlasmaCashRequestBatch{
			Requests: []*pctypes.PlasmaCashRequest{
				&pctypes.PlasmaCashRequest{
					Data: &pctypes.PlasmaCashRequest_Withdraw{&pctypes.PlasmaCashWithdrawCoinRequest{
						Owner: addr2.MarshalPB(),
						Slot:  coin.Slot,
					}},
					Meta: &pctypes.PlasmaCashEventMeta{
						BlockNumber: uint64(i) + 1,
						LogIndex:    0,
						TxIndex:     0,
					},
				},
			},
		})
		require.Nil(t, err)
	}
	resp, err := plasmaContract.BalanceOf(ctx, &BalanceOfRequest{
		Owner:    addr2.MarshalPB(),
		Contract: contractAddr.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Len(t, resp.Coins, 0)
}

func TestGetUserSlotsRequest(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)

	req2 := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:          8,
			NewOwner:      addr2.MarshalPB(),
			PreviousBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			Denomination:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
		},
	}
	generatedSender, err := setupPlasmaTxAuth(ctx, req2.Plasmatx, addr1)
	require.Nil(t, err)

	err = plasmaContract.ProcessRequestBatch(ctx, &pctypes.PlasmaCashRequestBatch{
		Requests: []*pctypes.PlasmaCashRequest{
			&pctypes.PlasmaCashRequest{
				Data: &pctypes.PlasmaCashRequest_Deposit{&pctypes.DepositRequest{
					Slot:         5,
					DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
					Denomination: &types.BigUInt{Value: *loom.NewBigUIntFromInt(100)},
					From:         addr2.MarshalPB(),
					Contract:     addr3.MarshalPB(),
				}},
				Meta: &pctypes.PlasmaCashEventMeta{
					BlockNumber: 1,
					LogIndex:    0,
					TxIndex:     0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = plasmaContract.ProcessRequestBatch(ctx, &pctypes.PlasmaCashRequestBatch{
		Requests: []*pctypes.PlasmaCashRequest{
			&pctypes.PlasmaCashRequest{
				Data: &pctypes.PlasmaCashRequest_Deposit{&pctypes.DepositRequest{
					Slot:         7,
					DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(4)},
					Denomination: &types.BigUInt{Value: *loom.NewBigUIntFromInt(200)},
					From:         addr2.MarshalPB(),
					Contract:     addr3.MarshalPB(),
				}},
				Meta: &pctypes.PlasmaCashEventMeta{
					BlockNumber: 2,
					LogIndex:    0,
					TxIndex:     0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = plasmaContract.ProcessRequestBatch(ctx, &pctypes.PlasmaCashRequestBatch{
		Requests: []*pctypes.PlasmaCashRequest{
			&pctypes.PlasmaCashRequest{
				Data: &pctypes.PlasmaCashRequest_Deposit{&pctypes.DepositRequest{
					Slot:         8,
					DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(5)},
					Denomination: &types.BigUInt{Value: *loom.NewBigUIntFromInt(200)},
					From:         generatedSender.MarshalPB(),
					Contract:     addr3.MarshalPB(),
				}},
				Meta: &pctypes.PlasmaCashEventMeta{
					BlockNumber: 3,
					LogIndex:    0,
					TxIndex:     0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = plasmaContract.PlasmaTxRequest(ctx, req2)
	require.Nil(t, err)

	reqMainnet := &SubmitBlockToMainnetRequest{}
	_, err = plasmaContract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	req := &GetUserSlotsRequest{
		From: addr2.MarshalPB(),
	}
	res, err := plasmaContract.GetUserSlotsRequest(ctx, req)
	require.Nil(t, err)

	assert.Equal(t, []uint64{5, 7, 8}, res.Slots, "slots should match")

}

func TestGetPlasmaTxRequestOnDepositBlock(t *testing.T) {
	plasmaContract, ctx := getPlasmaContractAndContext(t)

	err := plasmaContract.ProcessRequestBatch(ctx, &pctypes.PlasmaCashRequestBatch{
		Requests: []*pctypes.PlasmaCashRequest{
			&pctypes.PlasmaCashRequest{
				Data: &pctypes.PlasmaCashRequest_Deposit{&pctypes.DepositRequest{
					Slot:         123,
					DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
					Denomination: &types.BigUInt{Value: *loom.NewBigUIntFromInt(100)},
					From:         addr2.MarshalPB(),
					Contract:     addr3.MarshalPB(),
				}},
				Meta: &pctypes.PlasmaCashEventMeta{
					BlockNumber: 1,
					LogIndex:    0,
					TxIndex:     0,
				},
			},
		},
	})
	require.Nil(t, err)

	reqPlasmaTx := &GetPlasmaTxRequest{}
	reqPlasmaTx.BlockHeight = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(3),
	}
	reqPlasmaTx.Slot = 123

	res, err := plasmaContract.GetPlasmaTxRequest(ctx, reqPlasmaTx)
	require.Nil(t, err)

	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}, res.Plasmatx.Proof, "proof should match")
}

func TestGetPlasmaTxRequestNonInclusion(t *testing.T) {
	fakeCtx := plugin.CreateFakeContext(addr1, addr1)
	addressMapperAddress := fakeCtx.CreateContract(address_mapper.Contract)
	fakeCtx.RegisterContract("addressmapper", addressMapperAddress, addressMapperAddress)

	ctx := contractpb.WrapPluginContext(
		fakeCtx,
	)

	contract := &PlasmaCash{}
	err := contract.Init(ctx, &InitRequest{
		Oracle: addr1.MarshalPB(),
	})
	require.Nil(t, err)

	pending := &PendingTxs{}
	ctx.Get(pendingTXsKey, pending)
	assert.Equal(t, len(pending.Transactions), 0, "length should be zero")

	contractAddr := loom.RootAddress("eth")

	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     5,
		Contract: contractAddr.MarshalPB(),
	}))
	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     6,
		Contract: contractAddr.MarshalPB(),
	}))

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:          6,
			NewOwner:      addr3.MarshalPB(),
			PreviousBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			Denomination:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
		},
	}

	generatedSender, err := setupPlasmaTxAuth(ctx, req.Plasmatx, addr1)
	require.Nil(t, err)
	err = saveAccount(ctx, &Account{
		Owner: generatedSender.MarshalPB(),
		Slots: []uint64{5, 6},
	})
	require.Nil(t, err)

	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	reqMainnet := &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	reqPlasmaTx := &GetPlasmaTxRequest{}
	reqPlasmaTx.BlockHeight = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(1000),
	}
	reqPlasmaTx.Slot = 5

	res, err := contract.GetPlasmaTxRequest(ctx, reqPlasmaTx)
	require.Nil(t, err)

	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x6d, 0x2e, 0xfd, 0x44, 0xd0, 0xe7, 0x76, 0x5, 0x9d, 0xc0, 0x9c, 0xd4, 0x4, 0xb9, 0x62, 0x99, 0xea, 0x3b, 0xb3, 0x5c, 0xb7, 0xdf, 0xd1, 0xfc, 0xcf, 0xf, 0x78, 0x6a, 0x9e, 0xc3, 0xb4, 0xa7}, res.Plasmatx.Proof, "proof should match")
}

func TestGetPlasmaTxRequest(t *testing.T) {
	fakeCtx := plugin.CreateFakeContext(addr1, addr1)
	addressMapperAddress := fakeCtx.CreateContract(address_mapper.Contract)
	fakeCtx.RegisterContract("addressmapper", addressMapperAddress, addressMapperAddress)
	ctx := contractpb.WrapPluginContext(
		fakeCtx,
	)

	contract := &PlasmaCash{}
	err := contract.Init(ctx, &InitRequest{
		Oracle: addr1.MarshalPB(),
	})
	require.Nil(t, err)

	ethPrivKey, err := crypto.GenerateKey()
	require.Nil(t, err)

	pending := &PendingTxs{}
	ctx.Get(pendingTXsKey, pending)
	assert.Equal(t, len(pending.Transactions), 0, "length should be zero")

	contractAddr := loom.RootAddress("eth")

	// Make the block have 2 transactions
	// (if only 1 tx in block we are in the best case scenario where we get 8 0's)
	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     5,
		Contract: contractAddr.MarshalPB(),
	}))
	require.Nil(t, saveCoin(ctx, &Coin{
		Slot:     6,
		Contract: contractAddr.MarshalPB(),
	}))

	req := &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:          5,
			NewOwner:      addr3.MarshalPB(),
			PreviousBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			Denomination:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
		},
	}

	generatedSender, err := setupPlasmaTxAuthWithKey(ctx, ethPrivKey, req.Plasmatx, addr1)
	require.Nil(t, err)

	err = saveAccount(ctx, &Account{
		Owner: generatedSender.MarshalPB(),
		Slots: []uint64{5, 6},
	})
	require.Nil(t, err)

	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	req = &PlasmaTxRequest{
		Plasmatx: &PlasmaTx{
			Slot:          6,
			Sender:        addr2.MarshalPB(),
			NewOwner:      addr3.MarshalPB(),
			PreviousBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
			Denomination:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(0)},
		},
	}

	generatedSender, err = setupPlasmaTxAuthWithKey(ctx, ethPrivKey, req.Plasmatx, addr1)
	require.Nil(t, err)

	err = contract.PlasmaTxRequest(ctx, req)
	require.Nil(t, err)

	reqMainnet := &SubmitBlockToMainnetRequest{}
	_, err = contract.SubmitBlockToMainnet(ctx, reqMainnet)
	require.Nil(t, err)

	reqPlasmaTx := &GetPlasmaTxRequest{}
	reqPlasmaTx.BlockHeight = &types.BigUInt{
		Value: *loom.NewBigUIntFromInt(1000),
	}
	reqPlasmaTx.Slot = 5

	res, err := contract.GetPlasmaTxRequest(ctx, reqPlasmaTx)
	require.Nil(t, err)

	assert.Equal(t, []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x6d, 0x2e, 0xfd, 0x44, 0xd0, 0xe7, 0x76, 0x5, 0x9d, 0xc0, 0x9c, 0xd4, 0x4, 0xb9, 0x62, 0x99, 0xea, 0x3b, 0xb3, 0x5c, 0xb7, 0xdf, 0xd1, 0xfc, 0xcf, 0xf, 0x78, 0x6a, 0x9e, 0xc3, 0xb4, 0xa7}, res.Plasmatx.Proof, "proof should match")
}

func TestOracleChange(t *testing.T) {
	oldOracleAddress := addr1
	newOracleAddress := addr3

	fakeCtx := plugin.CreateFakeContext(oldOracleAddress, addr1)
	ctx := contractpb.WrapPluginContext(fakeCtx)

	tokenIDs := []*types.BigUInt{
		&types.BigUInt{Value: *loom.NewBigUIntFromInt(721)},
		&types.BigUInt{Value: *loom.NewBigUIntFromInt(127)},
	}

	plasmaContract := &PlasmaCash{}
	err := plasmaContract.Init(ctx, &InitRequest{
		Oracle: oldOracleAddress.MarshalPB(),
	})
	require.Nil(t, err)

	// Only oracle can appoint new oracle
	err = plasmaContract.UpdateOracle(ctx, &UpdateOracleRequest{
		NewOracle: newOracleAddress.MarshalPB(),
	})
	require.Nil(t, err)

	// Now, previous oracle wont work

	// Only current oracle can call DepositRequest
	err = plasmaContract.ProcessRequestBatch(contractpb.WrapPluginContext(fakeCtx.WithSender(oldOracleAddress)), &pctypes.PlasmaCashRequestBatch{
		Requests: []*pctypes.PlasmaCashRequest{
			&pctypes.PlasmaCashRequest{
				Data: &pctypes.PlasmaCashRequest_Deposit{&pctypes.DepositRequest{
					Slot:         123,
					DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
					Denomination: tokenIDs[0],
					From:         addr2.MarshalPB(),
					Contract:     addr3.MarshalPB(),
				}},
				Meta: &pctypes.PlasmaCashEventMeta{
					BlockNumber: 1,
					LogIndex:    0,
					TxIndex:     0,
				},
			},
		},
	})

	require.Equal(t, err, ErrNotAuthorized)

	// Only current oracle can appoint new oracle
	err = plasmaContract.UpdateOracle(contractpb.WrapPluginContext(fakeCtx.WithSender(oldOracleAddress)),
		&UpdateOracleRequest{
			NewOracle: addr3.MarshalPB(),
		})
	require.Equal(t, err, ErrNotAuthorized)

	// New oracle should work
	err = plasmaContract.ProcessRequestBatch(contractpb.WrapPluginContext(fakeCtx.WithSender(newOracleAddress)), &pctypes.PlasmaCashRequestBatch{
		Requests: []*pctypes.PlasmaCashRequest{
			&pctypes.PlasmaCashRequest{
				Data: &pctypes.PlasmaCashRequest_Deposit{&pctypes.DepositRequest{
					Slot:         123,
					DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
					Denomination: tokenIDs[0],
					From:         addr2.MarshalPB(),
					Contract:     addr3.MarshalPB(),
				}},
				Meta: &pctypes.PlasmaCashEventMeta{
					BlockNumber: 1,
					LogIndex:    0,
					TxIndex:     0,
				},
			},
		},
	})
	require.Nil(t, err)

	// New Oracle should able to appoint another oracle
	err = plasmaContract.UpdateOracle(contractpb.WrapPluginContext(fakeCtx.WithSender(newOracleAddress)),
		&UpdateOracleRequest{
			NewOracle: addr2.MarshalPB(),
		})
	require.Nil(t, err)

}

func TestOracleAuth(t *testing.T) {
	fakeCtx := plugin.CreateFakeContext(addr1, addr1)
	notAuthorizedCtx := contractpb.WrapPluginContext(fakeCtx)

	fakeCtx2 := plugin.CreateFakeContext(addr2, addr2)
	authorizedCtx := contractpb.WrapPluginContext(fakeCtx2)

	plasmaContract := &PlasmaCash{}
	err := plasmaContract.Init(authorizedCtx, &InitRequest{
		Oracle: addr2.MarshalPB(),
	})
	require.Nil(t, err)

	tokenIDs := []*types.BigUInt{
		&types.BigUInt{Value: *loom.NewBigUIntFromInt(721)},
		&types.BigUInt{Value: *loom.NewBigUIntFromInt(127)},
	}

	// Non oracle sender wont be able to call this method
	err = plasmaContract.ProcessRequestBatch(notAuthorizedCtx, &pctypes.PlasmaCashRequestBatch{
		Requests: []*pctypes.PlasmaCashRequest{
			&pctypes.PlasmaCashRequest{
				Data: &pctypes.PlasmaCashRequest_Deposit{&pctypes.DepositRequest{
					Slot:         123,
					DepositBlock: &types.BigUInt{Value: *loom.NewBigUIntFromInt(3)},
					Denomination: tokenIDs[0],
					From:         addr2.MarshalPB(),
					Contract:     addr3.MarshalPB(),
				}},
				Meta: &pctypes.PlasmaCashEventMeta{
					BlockNumber: 1,
					LogIndex:    0,
					TxIndex:     0,
				},
			},
		},
	})

	require.Equal(t, err, ErrNotAuthorized)

	// Non oracle cant update oracle
	err = plasmaContract.UpdateOracle(notAuthorizedCtx, &UpdateOracleRequest{
		NewOracle: addr1.MarshalPB(),
	})
	require.Equal(t, err, ErrNotAuthorized)

}

func setupPlasmaTxAuth(ctx contractpb.Context, plasmaTx *PlasmaTx, dappchainAddr loom.Address) (loom.Address, error) {
	ethPrivKey, err := crypto.GenerateKey()
	if err != nil {
		return loom.Address{}, err
	}

	return setupPlasmaTxAuthWithKey(ctx, ethPrivKey, plasmaTx, dappchainAddr)

}

func setupPlasmaTxAuthWithKey(ctx contractpb.Context, ethPrivKey *ecdsa.PrivateKey, plasmaTx *PlasmaTx, dappchainAddr loom.Address) (loom.Address, error) {
	hash, signature, err := getHashAndSignature(plasmaTx, ethPrivKey)
	if err != nil {
		return loom.Address{}, err
	}

	ethLocalAddress, err := evmcompat.RecoverAddressFromTypedSig(
		hash, signature, []evmcompat.SignatureType{evmcompat.SignatureType_EIP712},
	)
	if err != nil {
		return loom.Address{}, err
	}

	ethAddress := loom.MustParseAddress(fmt.Sprintf("eth:%s", ethLocalAddress.Hex()))

	plasmaTx.Signature = signature
	plasmaTx.Sender = ethAddress.MarshalPB()

	registerAddressMapping(ctx, dappchainAddr, ethAddress, ethPrivKey)
	if err != nil {
		return loom.Address{}, err
	}

	return ethAddress, nil
}

func registerAddressMapping(ctx contractpb.Context, from, to loom.Address, key *ecdsa.PrivateKey) error {
	addressMappingSig, err := generateAddressMappingSignature(from, to, key)
	if err != nil {
		return err
	}

	addressMapperAddress, err := ctx.Resolve("addressmapper")
	if err != nil {
		return err
	}

	return contractpb.CallMethod(ctx, addressMapperAddress, "AddIdentityMapping", &amtypes.AddressMapperAddIdentityMappingRequest{
		From:      from.MarshalPB(),
		To:        to.MarshalPB(),
		Signature: addressMappingSig,
	}, nil)
}

func generateAddressMappingSignature(from, to loom.Address, key *ecdsa.PrivateKey) ([]byte, error) {
	hash := ssha.SoliditySHA3(
		ssha.Address(ethcommon.BytesToAddress(from.Local)),
		ssha.Address(ethcommon.BytesToAddress(to.Local)),
	)
	sig, err := evmcompat.SoliditySign(hash, key)
	if err != nil {
		return nil, err
	}
	// Prefix the sig with a single byte indicating the sig type, in this case EIP712
	return append(make([]byte, 1, 66), sig...), nil
}

func getHashAndSignature(plasmatx *PlasmaTx, privateKey *ecdsa.PrivateKey) ([]byte, []byte, error) {
	loomTx := &plasma_cash.LoomTx{
		Slot:         plasmatx.Slot,
		Denomination: plasmatx.Denomination.Value.Int,
		Owner:        ethcommon.BytesToAddress(plasmatx.NewOwner.Local),
		PrevBlock:    plasmatx.PreviousBlock.Value.Int,
		TXProof:      plasmatx.Proof,
	}

	calculatedPlasmaTxHash, err := loomTx.Hash()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unable to calculate plasmaTx hash")
	}

	signature, err := evmcompat.GenerateTypedSig(calculatedPlasmaTxHash, privateKey, evmcompat.SignatureType_EIP712)
	if err != nil {
		return nil, nil, err
	}

	return calculatedPlasmaTxHash, signature, nil
}

func getPlasmaContractAndContext(t *testing.T) (*PlasmaCash, contractpb.Context) {
	fakeCtx := plugin.CreateFakeContext(addr1, addr1)
	addressMapperAddress := fakeCtx.CreateContract(address_mapper.Contract)
	fakeCtx.RegisterContract("addressmapper", addressMapperAddress, addressMapperAddress)
	ctx := contractpb.WrapPluginContext(fakeCtx)

	plasmaContract := &PlasmaCash{}
	err := plasmaContract.Init(ctx, &InitRequest{
		Oracle: addr1.MarshalPB(),
	})
	require.Nil(t, err)

	return plasmaContract, ctx
}
