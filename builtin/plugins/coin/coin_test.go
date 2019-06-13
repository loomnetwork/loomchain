package coin

import (
	"errors"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/loomnetwork/go-loom"
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
	pctx.SetFeature(loomchain.CoinPolicyFeature, true)
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
			DeflationFactorDenominator: 5,
			DeflationFactorNumerator:   1,
			MintingAccount:             addr1.MarshalPB(),
		},
		BaseMintingAmount: 50,
	})
	require.Nil(t, err)
	//Nil Invalid Policy
	err = contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy:            nil,
		BaseMintingAmount: 50,
	})
	require.Error(t, errors.New("Policy is not specified"), err.Error())
	//Invalid Policy Denominator == 0
	err = contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: &Policy{
			DeflationFactorDenominator: 0,
			DeflationFactorNumerator:   1,
			MintingAccount:             addr1.MarshalPB(),
		},
		BaseMintingAmount: 50,
	})
	require.Error(t, errors.New("DeflationFactorDenominator should be greater than zero"), err.Error())
	//Invalid Policy Numerator == 0
	err = contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: &Policy{
			DeflationFactorDenominator: 0,
			DeflationFactorNumerator:   0,
			MintingAccount:             addr1.MarshalPB(),
		},
		BaseMintingAmount: 50,
	})
	require.Error(t, errors.New("DeflationFactorNumerator should be greater than zero"), err.Error())
	//Invalid Policy Base Minting Amount == 0
	err = contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: &Policy{
			DeflationFactorDenominator: 1,
			DeflationFactorNumerator:   5,
			MintingAccount:             addr1.MarshalPB(),
		},
		BaseMintingAmount: 0,
	})
	require.EqualError(t, errors.New("Base Minting Amount should be greater than zero"), err.Error())
	//Invalid Policy Invalid Minting Account
	err = contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			&InitialAccount{
				Owner:   addr1.MarshalPB(),
				Balance: uint64(31),
			},
		},
		Policy: &Policy{
			DeflationFactorDenominator: 1,
			DeflationFactorNumerator:   5,
			MintingAccount:             loom.RootAddress("chain").MarshalPB(),
		},
		BaseMintingAmount: 50,
	})
	require.Error(t, errors.New("Minting Account Address cannot be Root Address"), err.Error())
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
