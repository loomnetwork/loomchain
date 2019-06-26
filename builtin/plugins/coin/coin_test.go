package coin

import (
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
)

var (
	addr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	addr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

type mockLoomCoinGateway struct {
}

func (m *mockLoomCoinGateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "loomcoin-gateway",
		Version: "0.1.0",
	}, nil
}

func (m *mockLoomCoinGateway) DummyMethod(ctx contractpb.Context, req *MintToGatewayRequest) error {
	return nil
}

func TestLoadPolicy(t *testing.T) {
	contract := &Coin{}
	pctx := plugin.CreateFakeContext(addr1, addr1)
	ctx := contractpb.WrapPluginContext(pctx)
	//Valid Policy
	err := contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: &Policy{
			ChangeRatioDenominator: 5,
			ChangeRatioNumerator:   1,
			MintingAccount:         addr1.MarshalPB(),
			BlocksGeneratedPerYear: 50000,
			TotalSupply:            100000,
		},
	})
	require.Nil(t, err)
	//Invalid Policy Denominator == 0
	err = contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: &Policy{
			ChangeRatioDenominator: 0,
			ChangeRatioNumerator:   1,
			MintingAccount:         addr1.MarshalPB(),
			BlocksGeneratedPerYear: 50000,
			TotalSupply:            100000,
		},
	})
	require.Error(t, errors.New("ChangeRatioDenominator should be greater than zero"), err.Error())
	//Invalid Policy Numerator == 0
	err = contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: &Policy{
			ChangeRatioDenominator: 5,
			ChangeRatioNumerator:   0,
			MintingAccount:         addr1.MarshalPB(),
			BlocksGeneratedPerYear: 50000,
			TotalSupply:            100000,
		},
	})
	require.Error(t, errors.New("ChangeRatioNumerator should be greater than zero"), err.Error())
	//Invalid Policy Base Minting Amount == 0
	err = contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: &Policy{
			ChangeRatioDenominator: 5,
			ChangeRatioNumerator:   1,
			MintingAccount:         addr1.MarshalPB(),
			BlocksGeneratedPerYear: 50000,
			TotalSupply:            0,
		},
	})
	require.EqualError(t, errors.New("Total Supply should be greater than zero"), err.Error())
	//Invalid Policy Invalid Minting Account
	err = contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: &Policy{
			ChangeRatioDenominator: 5,
			ChangeRatioNumerator:   1,
			MintingAccount:         addr1.MarshalPB(),
			BlocksGeneratedPerYear: 0,
			TotalSupply:            100000,
		},
	})
	require.EqualError(t, errors.New("Blocks Generated Per Year should be greater than zero"), err.Error())
	//Invalid Policy Invalid Minting Account
	err = contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: &Policy{
			ChangeRatioDenominator: 5,
			ChangeRatioNumerator:   0,
			MintingAccount:         loom.RootAddress("chain").MarshalPB(),
			BlocksGeneratedPerYear: 0,
			TotalSupply:            100000,
		},
	})
	require.Error(t, errors.New("Minting Account Address cannot be Root Address"), err.Error())
}

func TestMint(t *testing.T) {
	//Initializing context for CoinPolicyFeature
	policy := &Policy{
		ChangeRatioDenominator: 5,
		ChangeRatioNumerator:   1,
		MintingAccount:         addr1.MarshalPB(),
		BlocksGeneratedPerYear: 50000,
		TotalSupply:            10000000,
	}
	pctx := plugin.CreateFakeContext(addr1, addr1)
	pctx.SetFeature(loomchain.CoinVersion1_2Feature, true)
	ctx := contractpb.WrapPluginContext(pctx.WithBlock(loom.BlockHeader{
		ChainID: "default",
		Time:    time.Now().Unix(),
		Height:  10000,
	}))
	//Minting will start in year 1 after BlockHeight 10000
	contract := &Coin{}
	err := contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: policy,
	})
	require.Nil(t, err)

	//Minting without any error
	resp1, err := contract.BalanceOf(ctx,
		&BalanceOfRequest{
			Owner: addr1.MarshalPB(),
		})
	require.Nil(t, err)
	err = Mint(ctx)
	require.Nil(t, err)

	// checking balance after minting
	resp2, err := contract.BalanceOf(ctx,
		&BalanceOfRequest{
			Owner: addr1.MarshalPB(),
		})
	require.Nil(t, err)

	var amount *common.BigUInt
	// Balance of mintingAccount should increase by specific amount after minting operation
	changeRatioNumerator := loom.NewBigUIntFromInt(int64(policy.ChangeRatioNumerator))
	changeRatioDenominator := loom.NewBigUIntFromInt(int64(policy.ChangeRatioDenominator))
	blockHeight := loom.NewBigUIntFromInt(ctx.Block().Height)
	totalSupply := loom.NewBigUIntFromInt(int64(policy.TotalSupply))
	blocksGeneratedPerYear := loom.NewBigUIntFromInt(int64(policy.BlocksGeneratedPerYear))
	year := blockHeight.Div(blockHeight, blocksGeneratedPerYear)
	//Computes year based on blockheight ==> year - 1
	assert.Equal(t, uint64(0), year.Uint64())
	if year == loom.NewBigUIntFromInt(0) {
		amount = totalSupply.Div(totalSupply, blocksGeneratedPerYear)
	} else {
		changeRatioNumerator = changeRatioNumerator.Exp(changeRatioNumerator, year, nil)
		changeRatioDenominator = changeRatioDenominator.Exp(changeRatioDenominator, year, nil)
		totalSupplyForYear := totalSupply.Mul(totalSupply, changeRatioNumerator)
		totalSupplyForYear = totalSupplyForYear.Div(totalSupplyForYear, changeRatioDenominator)
		amount = totalSupplyForYear.Div(totalSupplyForYear, blocksGeneratedPerYear)
	}
	// Minting starts for year 1 after blockheight 10000 - Minting Amount Per Block = 10000000*(1/50000) = 200
	assert.Equal(t, amount.Uint64(), resp2.Balance.Value.Uint64()-resp1.Balance.Value.Uint64())

	pctx1 := plugin.CreateFakeContext(addr1, addr1)
	pctx1.SetFeature(loomchain.CoinVersion1_2Feature, true)
	ctx1 := contractpb.WrapPluginContext(pctx1.WithBlock(loom.BlockHeader{
		ChainID: "default",
		Time:    time.Now().Unix(),
		Height:  60000,
	}))
	//Minting will start in year 2 after BlockHeight 60000
	contract1 := &Coin{}
	err1 := contract1.Init(ctx1, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: policy,
	})
	require.Nil(t, err1)

	//Minting without any error
	resp3, err1 := contract1.BalanceOf(ctx1,
		&BalanceOfRequest{
			Owner: addr1.MarshalPB(),
		})
	require.Nil(t, err1)
	err1 = Mint(ctx1)
	require.Nil(t, err1)

	// checking balance after minting
	resp4, err1 := contract1.BalanceOf(ctx1,
		&BalanceOfRequest{
			Owner: addr1.MarshalPB(),
		})
	require.Nil(t, err1)

	// Balance of mintingAccount should increase by specific amount after minting operation
	changeRatioNumerator = loom.NewBigUIntFromInt(int64(policy.ChangeRatioNumerator))
	changeRatioDenominator = loom.NewBigUIntFromInt(int64(policy.ChangeRatioDenominator))
	blockHeight = loom.NewBigUIntFromInt(ctx1.Block().Height)
	totalSupply = loom.NewBigUIntFromInt(int64(policy.TotalSupply))
	blocksGeneratedPerYear = loom.NewBigUIntFromInt(int64(policy.BlocksGeneratedPerYear))
	year = blockHeight.Div(blockHeight, blocksGeneratedPerYear)
	//Computes year based on blockheight year ==> 2
	assert.Equal(t, uint64(1), year.Uint64())
	if year == loom.NewBigUIntFromInt(0) {
		amount = totalSupply.Div(totalSupply, blocksGeneratedPerYear)
	} else {
		changeRatioNumerator = changeRatioNumerator.Exp(changeRatioNumerator, year, nil)
		changeRatioDenominator = changeRatioDenominator.Exp(changeRatioDenominator, year, nil)
		totalSupplyForYear := totalSupply.Mul(totalSupply, changeRatioNumerator)
		totalSupplyForYear = totalSupplyForYear.Div(totalSupplyForYear, changeRatioDenominator)
		amount = totalSupplyForYear.Div(totalSupplyForYear, blocksGeneratedPerYear)
	}
	// Minting starts for year 2 after blockheight 60000 - Minting Amount Per Block = 10000000*(1/5)*(1/50000) = 40
	assert.Equal(t, amount.Uint64(), resp4.Balance.Value.Uint64()-resp3.Balance.Value.Uint64())

	pctx2 := plugin.CreateFakeContext(addr1, addr1)
	pctx2.SetFeature(loomchain.CoinVersion1_2Feature, true)
	ctx2 := contractpb.WrapPluginContext(pctx2.WithBlock(loom.BlockHeader{
		ChainID: "default",
		Time:    time.Now().Unix(),
		Height:  110000,
	}))
	//Minting will start in year 3 after BlockHeight 110000
	contract2 := &Coin{}
	err2 := contract2.Init(ctx2, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: policy,
	})
	require.Nil(t, err2)

	//Minting without any error
	resp5, err2 := contract2.BalanceOf(ctx2,
		&BalanceOfRequest{
			Owner: addr1.MarshalPB(),
		})
	require.Nil(t, err2)
	err2 = Mint(ctx2)
	require.Nil(t, err2)

	// checking balance after minting
	resp6, err2 := contract2.BalanceOf(ctx2,
		&BalanceOfRequest{
			Owner: addr1.MarshalPB(),
		})
	require.Nil(t, err2)

	// Balance of mintingAccount should increase by specific amount after minting operation
	changeRatioNumerator = loom.NewBigUIntFromInt(int64(policy.ChangeRatioNumerator))
	changeRatioDenominator = loom.NewBigUIntFromInt(int64(policy.ChangeRatioDenominator))
	blockHeight = loom.NewBigUIntFromInt(ctx2.Block().Height)
	totalSupply = loom.NewBigUIntFromInt(int64(policy.TotalSupply))
	blocksGeneratedPerYear = loom.NewBigUIntFromInt(int64(policy.BlocksGeneratedPerYear))
	year = blockHeight.Div(blockHeight, blocksGeneratedPerYear)
	//Computes year based on blockheight year ==> 3
	assert.Equal(t, uint64(2), year.Uint64())
	if year == loom.NewBigUIntFromInt(0) {
		amount = totalSupply.Div(totalSupply, blocksGeneratedPerYear)
	} else {
		changeRatioNumerator = changeRatioNumerator.Exp(changeRatioNumerator, year, nil)
		changeRatioDenominator = changeRatioDenominator.Exp(changeRatioDenominator, year, nil)
		totalSupplyForYear := totalSupply.Mul(totalSupply, changeRatioNumerator)
		totalSupplyForYear = totalSupplyForYear.Div(totalSupplyForYear, changeRatioDenominator)
		amount = totalSupplyForYear.Div(totalSupplyForYear, blocksGeneratedPerYear)
	}
	// Minting starts for year 3 after blockheight 110000 - Minting Amount per block = 10000000*(1/5)*(1/5)*(1/50000) = 8
	assert.Equal(t, amount.Uint64(), resp6.Balance.Value.Uint64()-resp5.Balance.Value.Uint64())

	pctx3 := plugin.CreateFakeContext(addr1, addr1)
	pctx3.SetFeature(loomchain.CoinVersion1_2Feature, true)
	ctx3 := contractpb.WrapPluginContext(pctx3.WithBlock(loom.BlockHeader{
		ChainID: "default",
		Time:    time.Now().Unix(),
		Height:  10000002,
	}))
	//Minting will stop at this stage as minting Amount per block = 0 after very long period i.e 200 years
	contract3 := &Coin{}
	err3 := contract3.Init(ctx3, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: policy,
	})
	require.Nil(t, err3)

	//Minting without any error
	resp7, err3 := contract3.BalanceOf(ctx3,
		&BalanceOfRequest{
			Owner: addr1.MarshalPB(),
		})
	require.Nil(t, err3)
	//There will be no minting at this stage as amount to mint per block becomes zero
	err3 = Mint(ctx3)
	require.Nil(t, err3)

	// checking balance after minting
	resp8, err3 := contract3.BalanceOf(ctx3,
		&BalanceOfRequest{
			Owner: addr1.MarshalPB(),
		})
	require.Nil(t, err3)

	// Balance of mintingAccount should increase by specific amount after minting operation
	changeRatioNumerator = loom.NewBigUIntFromInt(int64(policy.ChangeRatioNumerator))
	changeRatioDenominator = loom.NewBigUIntFromInt(int64(policy.ChangeRatioDenominator))
	blockHeight = loom.NewBigUIntFromInt(ctx3.Block().Height)
	totalSupply = loom.NewBigUIntFromInt(int64(policy.TotalSupply))
	blocksGeneratedPerYear = loom.NewBigUIntFromInt(int64(policy.BlocksGeneratedPerYear))
	year = blockHeight.Div(blockHeight, blocksGeneratedPerYear)
	//Year comes out to be very long period i.e 200 years
	assert.Equal(t, uint64(200), year.Uint64())

	if year == loom.NewBigUIntFromInt(0) {
		amount = totalSupply.Div(totalSupply, blocksGeneratedPerYear)
	} else {
		changeRatioNumerator = changeRatioNumerator.Exp(changeRatioNumerator, year, nil)
		changeRatioDenominator = changeRatioDenominator.Exp(changeRatioDenominator, year, nil)
		totalSupplyForYear := totalSupply.Mul(totalSupply, changeRatioNumerator)
		totalSupplyForYear = totalSupplyForYear.Div(totalSupplyForYear, changeRatioDenominator)
		amount = totalSupplyForYear.Div(totalSupplyForYear, blocksGeneratedPerYear)
	}
	// Minting stops at this stage and total supply becomes constant
	assert.Equal(t, amount.Uint64(), resp8.Balance.Value.Uint64()-resp7.Balance.Value.Uint64())
	assert.Equal(t, amount.Uint64(), uint64(0))

}

func TestTransfer(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	amount := loom.NewBigUIntFromInt(100)
	contract := &Coin{}
	err := contract.Transfer(ctx, &TransferRequest{
		To:     addr2.MarshalPB(),
		Amount: &types.BigUInt{Value: *amount},
	})
	assert.NotNil(t, err)

	acct := &Account{
		Owner: addr1.MarshalPB(),
		Balance: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(100),
		},
	}
	err = saveAccount(ctx, acct)
	require.Nil(t, err)

	err = contract.Transfer(ctx, &TransferRequest{
		To:     addr2.MarshalPB(),
		Amount: &types.BigUInt{Value: *amount},
	})
	assert.Nil(t, err)

	resp, err := contract.BalanceOf(ctx, &BalanceOfRequest{
		Owner: addr1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, 0, int(resp.Balance.Value.Int64()))

	resp, err = contract.BalanceOf(ctx, &BalanceOfRequest{
		Owner: addr2.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, 100, int(resp.Balance.Value.Int64()))
}

func sciNot(m, n int64) *loom.BigUInt {
	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(n), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(m))
	return ret
}

// Verify Coin.Transfer works correctly when the to & from addresses are the same.
func TestTransferToSelf(t *testing.T) {
	pctx := plugin.CreateFakeContext(addr1, addr1)
	// Test using the v1.1 contract, this test will fail if this feature is not enabled
	pctx.SetFeature(loomchain.CoinVersion1_1Feature, true)

	contract := &Coin{}
	err := contract.Init(
		contractpb.WrapPluginContext(pctx),
		&InitRequest{
			Accounts: []*InitialAccount{
				&InitialAccount{
					Owner:   addr2.MarshalPB(),
					Balance: uint64(100),
				},
			},
		},
	)
	require.NoError(t, err)

	amount := sciNot(100, 18)
	resp, err := contract.BalanceOf(
		contractpb.WrapPluginContext(pctx),
		&BalanceOfRequest{
			Owner: addr2.MarshalPB(),
		},
	)
	require.NoError(t, err)
	assert.Equal(t, *amount, resp.Balance.Value)

	err = contract.Transfer(
		contractpb.WrapPluginContext(pctx.WithSender(addr2)),
		&TransferRequest{
			To:     addr2.MarshalPB(),
			Amount: &types.BigUInt{Value: *amount},
		},
	)
	assert.NoError(t, err)

	resp, err = contract.BalanceOf(
		contractpb.WrapPluginContext(pctx),
		&BalanceOfRequest{
			Owner: addr2.MarshalPB(),
		},
	)
	require.NoError(t, err)
	// the transfer was from addr2 to addr2 so the balance of addr2 should remain unchanged
	assert.Equal(t, *amount, resp.Balance.Value)
}

func TestApprove(t *testing.T) {
	contract := &Coin{}

	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)
	acct := &Account{
		Owner: addr1.MarshalPB(),
		Balance: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(100),
		},
	}
	err := saveAccount(ctx, acct)
	require.Nil(t, err)

	err = contract.Approve(ctx, &ApproveRequest{
		Spender: addr3.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(40),
		},
	})
	assert.Nil(t, err)

	allowResp, err := contract.Allowance(ctx, &AllowanceRequest{
		Owner:   addr1.MarshalPB(),
		Spender: addr3.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, 40, int(allowResp.Amount.Value.Int64()))
}

func TestTransferFrom(t *testing.T) {
	contract := &Coin{}

	pctx := plugin.CreateFakeContext(addr1, addr1)
	ctx := contractpb.WrapPluginContext(pctx)
	acct := &Account{
		Owner: addr1.MarshalPB(),
		Balance: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(100),
		},
	}
	err := saveAccount(ctx, acct)
	require.Nil(t, err)

	err = contract.Approve(ctx, &ApproveRequest{
		Spender: addr3.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(40),
		},
	})
	assert.Nil(t, err)

	ctx = contractpb.WrapPluginContext(pctx.WithSender(addr3))
	err = contract.TransferFrom(ctx, &TransferFromRequest{
		From: addr1.MarshalPB(),
		To:   addr2.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(50),
		},
	})
	assert.NotNil(t, err)

	err = contract.TransferFrom(ctx, &TransferFromRequest{
		From: addr1.MarshalPB(),
		To:   addr2.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(30),
		},
	})
	assert.Nil(t, err)

	allowResp, err := contract.Allowance(ctx, &AllowanceRequest{
		Owner:   addr1.MarshalPB(),
		Spender: addr3.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, 10, int(allowResp.Amount.Value.Int64()))

	balResp, err := contract.BalanceOf(ctx, &BalanceOfRequest{
		Owner: addr1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, 70, int(balResp.Balance.Value.Int64()))

	balResp, err = contract.BalanceOf(ctx, &BalanceOfRequest{
		Owner: addr2.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, 30, int(balResp.Balance.Value.Int64()))
}

// Verify Coin.TransferFrom works correctly when the to & from addresses are the same.
func TestTransferFromSelf(t *testing.T) {
	pctx := plugin.CreateFakeContext(addr1, addr1)
	// Test using the v1.1 contract, this test will fail if this feature is not enabled
	pctx.SetFeature(loomchain.CoinVersion1_1Feature, true)

	contract := &Coin{}
	err := contract.Init(
		contractpb.WrapPluginContext(pctx),
		&InitRequest{
			Accounts: []*InitialAccount{
				&InitialAccount{
					Owner:   addr2.MarshalPB(),
					Balance: uint64(100),
				},
			},
		},
	)
	require.NoError(t, err)

	amount := sciNot(100, 18)
	resp, err := contract.BalanceOf(
		contractpb.WrapPluginContext(pctx),
		&BalanceOfRequest{
			Owner: addr2.MarshalPB(),
		},
	)
	require.NoError(t, err)
	assert.Equal(t, *amount, resp.Balance.Value)

	err = contract.Approve(
		contractpb.WrapPluginContext(pctx.WithSender(addr2)),
		&ApproveRequest{
			Spender: addr2.MarshalPB(),
			Amount: &types.BigUInt{
				Value: *amount,
			},
		},
	)
	assert.NoError(t, err)

	err = contract.TransferFrom(
		contractpb.WrapPluginContext(pctx.WithSender(addr2)),
		&TransferFromRequest{
			From: addr2.MarshalPB(),
			To:   addr2.MarshalPB(),
			Amount: &types.BigUInt{
				Value: *amount,
			},
		},
	)
	assert.NoError(t, err)

	resp, err = contract.BalanceOf(
		contractpb.WrapPluginContext(pctx),
		&BalanceOfRequest{
			Owner: addr2.MarshalPB(),
		},
	)
	require.NoError(t, err)
	// the transfer was from addr2 to addr2 so the balance of addr2 should remain unchanged
	assert.Equal(t, *amount, resp.Balance.Value)
}

func TestMintToGateway(t *testing.T) {
	contract := &Coin{}

	mockLoomCoinGatewayContract := contractpb.MakePluginContract(&mockLoomCoinGateway{})

	pctx := plugin.CreateFakeContext(addr1, addr1)

	loomcoinTGAddress := pctx.CreateContract(mockLoomCoinGatewayContract)
	pctx.RegisterContract("loomcoin-gateway", loomcoinTGAddress, loomcoinTGAddress)

	ctx := contractpb.WrapPluginContext(pctx)

	contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   loomcoinTGAddress.MarshalPB(),
				Balance: uint64(29),
			},
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
	})

	multiplier := big.NewInt(10).Exp(big.NewInt(10), big.NewInt(18), big.NewInt(0))
	loomcoinTGBalance := big.NewInt(0).Mul(multiplier, big.NewInt(29))
	addr1Balance := big.NewInt(0).Mul(multiplier, big.NewInt(31))
	totalSupply := big.NewInt(0).Add(loomcoinTGBalance, addr1Balance)

	totalSupplyResponse, err := contract.TotalSupply(ctx, &TotalSupplyRequest{})
	require.Nil(t, err)
	require.Equal(t, totalSupply, totalSupplyResponse.TotalSupply.Value.Int)

	gatewayBalnanceResponse, err := contract.BalanceOf(ctx, &BalanceOfRequest{
		Owner: loomcoinTGAddress.MarshalPB(),
	})
	require.Nil(t, err)
	require.Equal(t, loomcoinTGBalance, gatewayBalnanceResponse.Balance.Value.Int)

	require.Nil(t, contract.MintToGateway(
		contractpb.WrapPluginContext(pctx.WithSender(loomcoinTGAddress)),
		&MintToGatewayRequest{
			Amount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(59),
			},
		},
	))

	newTotalSupply := big.NewInt(0).Add(totalSupply, big.NewInt(59))
	newLoomCoinTGBalance := big.NewInt(0).Add(loomcoinTGBalance, big.NewInt(59))

	totalSupplyResponse, err = contract.TotalSupply(ctx, &TotalSupplyRequest{})
	require.Nil(t, err)
	require.Equal(t, newTotalSupply, totalSupplyResponse.TotalSupply.Value.Int)

	gatewayBalnanceResponse, err = contract.BalanceOf(ctx, &BalanceOfRequest{
		Owner: loomcoinTGAddress.MarshalPB(),
	})
	require.Nil(t, err)
	require.Equal(t, newLoomCoinTGBalance, gatewayBalnanceResponse.Balance.Value.Int)
}

func TestBurn(t *testing.T) {
	contract := &Coin{}

	mockLoomCoinGatewayContract := contractpb.MakePluginContract(&mockLoomCoinGateway{})

	pctx := plugin.CreateFakeContext(addr1, addr1)

	loomcoinTGAddress := pctx.CreateContract(mockLoomCoinGatewayContract)
	pctx.RegisterContract("loomcoin-gateway", loomcoinTGAddress, loomcoinTGAddress)

	ctx := contractpb.WrapPluginContext(pctx)

	contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr2.MarshalPB(),
				Balance: uint64(29),
			},
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
	})

	multiplier := big.NewInt(10).Exp(big.NewInt(10), big.NewInt(18), big.NewInt(0))
	addr2Balance := big.NewInt(0).Mul(multiplier, big.NewInt(29))
	addr1Balance := big.NewInt(0).Mul(multiplier, big.NewInt(31))
	totalSupply := big.NewInt(0).Add(addr2Balance, addr1Balance)

	totalSupplyResponse, err := contract.TotalSupply(ctx, &TotalSupplyRequest{})
	require.Nil(t, err)
	require.Equal(t, totalSupply, totalSupplyResponse.TotalSupply.Value.Int)

	addr2BalanceResponse, err := contract.BalanceOf(ctx, &BalanceOfRequest{
		Owner: addr2.MarshalPB(),
	})

	require.Nil(t, err)
	require.Equal(t, addr2Balance, addr2BalanceResponse.Balance.Value.Int)

	require.Nil(t, contract.Burn(
		contractpb.WrapPluginContext(pctx.WithSender(loomcoinTGAddress)),
		&BurnRequest{
			Owner: addr2.MarshalPB(),
			Amount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(2),
			},
		},
	))

	newTotalSupply := big.NewInt(0).Sub(totalSupply, big.NewInt(2))
	newAddr2Balance := big.NewInt(0).Sub(addr2Balance, big.NewInt(2))

	totalSupplyResponse, err = contract.TotalSupply(ctx, &TotalSupplyRequest{})
	require.Nil(t, err)
	require.Equal(t, newTotalSupply, totalSupplyResponse.TotalSupply.Value.Int)

	addr2BalanceResponse, err = contract.BalanceOf(ctx, &BalanceOfRequest{
		Owner: addr2.MarshalPB(),
	})
	require.Nil(t, err)
	require.Equal(t, newAddr2Balance, addr2BalanceResponse.Balance.Value.Int)
}

func TestBurnAccess(t *testing.T) {
	contract := &Coin{}

	mockLoomCoinGatewayContract := contractpb.MakePluginContract(&mockLoomCoinGateway{})

	pctx := plugin.CreateFakeContext(addr1, addr1)

	loomcoinTGAddress := pctx.CreateContract(mockLoomCoinGatewayContract)
	pctx.RegisterContract("loomcoin-gateway", loomcoinTGAddress, loomcoinTGAddress)

	ctx := contractpb.WrapPluginContext(pctx)

	contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			{
				Owner:   addr1.MarshalPB(),
				Balance: 100,
			},
			{
				Owner:   addr2.MarshalPB(),
				Balance: 0,
			},
		},
	})

	require.EqualError(t, contract.Burn(ctx, &BurnRequest{
		Owner: addr1.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(10),
		},
	}), "not authorized to burn Loom coin", "only loomcoin gateway can call Burn")

	require.Nil(t, contract.Burn(
		contractpb.WrapPluginContext(pctx.WithSender(loomcoinTGAddress)),
		&BurnRequest{
			Owner: addr1.MarshalPB(),
			Amount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(10),
			},
		},
	), "loomcoin gateway should be allowed to call Burn")

	require.EqualError(t, contract.Burn(
		contractpb.WrapPluginContext(pctx.WithSender(loomcoinTGAddress)),
		&BurnRequest{
			Owner: addr2.MarshalPB(),
			Amount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(10),
			},
		},
	), "cant burn coins more than available balance: 0", "only burn coin owned by you")
}

func TestMintToGatewayAccess(t *testing.T) {
	contract := &Coin{}

	mockLoomCoinGatewayContract := contractpb.MakePluginContract(&mockLoomCoinGateway{})

	pctx := plugin.CreateFakeContext(addr1, addr1)

	loomcoinTGAddress := pctx.CreateContract(mockLoomCoinGatewayContract)
	pctx.RegisterContract("loomcoin-gateway", loomcoinTGAddress, loomcoinTGAddress)

	ctx := contractpb.WrapPluginContext(pctx)

	require.EqualError(t, contract.MintToGateway(ctx, &MintToGatewayRequest{
		Amount: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(10),
		},
	}), "not authorized to mint Loom coin", "only loomcoin gateway can call MintToGateway")

	require.Nil(t, contract.MintToGateway(
		contractpb.WrapPluginContext(pctx.WithSender(loomcoinTGAddress)),
		&MintToGatewayRequest{
			Amount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(10),
			},
		},
	), "loomcoin gateway should be allowed to call MintToGateway")

}
