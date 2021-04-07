package coin

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/features"
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

type mockBinanceGateway struct {
}

func (m *mockBinanceGateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "binance-gateway",
		Version: "0.1.0",
	}, nil
}

func (m *mockBinanceGateway) DummyMethod(ctx contractpb.Context, req *MintToGatewayRequest) error {
	return nil
}

type mockBscGateway struct {
}

func (m *mockBscGateway) Meta() (plugin.Meta, error) {
	return plugin.Meta{
		Name:    "bsc-gateway",
		Version: "0.1.0",
	}, nil
}

func (m *mockBscGateway) DummyMethod(ctx contractpb.Context, req *MintToGatewayRequest) error {
	return nil
}

func TestTransfer(t *testing.T) {
	pctx := plugin.CreateFakeContext(addr1, addr2)
	ctx := contractpb.WrapPluginContext(pctx)

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

	pctx.SetFeature(features.CoinVersion1_2Feature, true)
	err = contract.Transfer(contractpb.WrapPluginContext(pctx), &TransferRequest{
		To:     nil,
		Amount: nil,
	})
	assert.Equal(t, ErrInvalidRequest, err)

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
	pctx.SetFeature(features.CoinVersion1_1Feature, true)

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
	pctx := plugin.CreateFakeContext(addr1, addr2)
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

	allowResp, err := contract.Allowance(ctx, &AllowanceRequest{
		Owner:   addr1.MarshalPB(),
		Spender: addr3.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, 40, int(allowResp.Amount.Value.Int64()))

	pctx.SetFeature(features.CoinVersion1_2Feature, true)
	err = contract.Approve(contractpb.WrapPluginContext(pctx), &ApproveRequest{
		Spender: nil,
		Amount:  nil,
	})
	assert.Equal(t, ErrInvalidRequest, err)
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

	pctx.SetFeature(features.CoinVersion1_2Feature, true)
	nilResp, err := contract.BalanceOf(ctx, &BalanceOfRequest{
		Owner: nil,
	})
	require.Error(t, err)
	require.Nil(t, nilResp)

}

// Verify Coin.TransferFrom works correctly when the to & from addresses are the same.
func TestTransferFromSelf(t *testing.T) {
	pctx := plugin.CreateFakeContext(addr1, addr1)
	// Test using the v1.1 contract, this test will fail if this feature is not enabled
	pctx.SetFeature(features.CoinVersion1_1Feature, true)

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

	pctx.SetFeature(features.CoinVersion1_2Feature, true)
	err = contract.MintToGateway(contractpb.WrapPluginContext(pctx.WithSender(loomcoinTGAddress)), &MintToGatewayRequest{
		Amount: nil,
	})
	assert.Equal(t, ErrInvalidRequest, err)
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
	mockBinanceGatewayContract := contractpb.MakePluginContract(&mockBinanceGateway{})
	mockBscGatewayContract := contractpb.MakePluginContract(&mockBscGateway{})

	pctx := plugin.CreateFakeContext(addr1, addr1)

	loomcoinTGAddress := pctx.CreateContract(mockLoomCoinGatewayContract)
	pctx.RegisterContract("loomcoin-gateway", loomcoinTGAddress, loomcoinTGAddress)
	binanceTGAddress := pctx.CreateContract(mockBinanceGatewayContract)
	pctx.RegisterContract("binance-gateway", binanceTGAddress, binanceTGAddress)
	bscTGAddress := pctx.CreateContract(mockBscGatewayContract)
	pctx.RegisterContract("bsc-gateway", bscTGAddress, bscTGAddress)

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
	}), "failed to burn LOOM: not authorized", "only gateway can call Burn")

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
	), "can't burn more coins than the available balance: 0", "only burn coin owned by you")

	require.EqualError(t, contract.Burn(
		contractpb.WrapPluginContext(pctx.WithSender(binanceTGAddress)),
		&BurnRequest{
			Owner: addr1.MarshalPB(),
			Amount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(10),
			},
		}),
		"failed to burn LOOM: not authorized",
		"Binance gateway shouldn't be allowed to burn if coin:v1.3 is disabled",
	)

	pctx.SetFeature(features.CoinVersion1_3Feature, true)
	require.Nil(t, contract.Burn(
		contractpb.WrapPluginContext(pctx.WithSender(binanceTGAddress)),
		&BurnRequest{
			Owner: addr1.MarshalPB(),
			Amount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(10),
			},
		},
	), "Binance gateway should be allowed to call MintToGateway")

	//TODO test a mint / burn on binance

	//TODO test mint / burn, on binance then eth

	//TODO test mint / burn, on eth then binance

}

func TestMintToGatewayAccess(t *testing.T) {
	contract := &Coin{}

	mockLoomCoinGatewayContract := contractpb.MakePluginContract(&mockLoomCoinGateway{})
	mockBinanceGatewayContract := contractpb.MakePluginContract(&mockBinanceGateway{})

	pctx := plugin.CreateFakeContext(addr1, addr1)

	loomcoinTGAddress := pctx.CreateContract(mockLoomCoinGatewayContract)
	pctx.RegisterContract("loomcoin-gateway", loomcoinTGAddress, loomcoinTGAddress)
	binanceTGAddress := pctx.CreateContract(mockBinanceGatewayContract)
	pctx.RegisterContract("binance-gateway", binanceTGAddress, binanceTGAddress)

	ctx := contractpb.WrapPluginContext(pctx)

	require.EqualError(t, contract.MintToGateway(ctx, &MintToGatewayRequest{
		Amount: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(10),
		},
	}), "failed to mint LOOM: not authorized", "only gateway can call MintToGateway")

	require.Nil(t, contract.MintToGateway(
		contractpb.WrapPluginContext(pctx.WithSender(loomcoinTGAddress)),
		&MintToGatewayRequest{
			Amount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(10),
			},
		},
	), "LOOM gateway should be allowed to call MintToGateway")

	require.EqualError(t, contract.MintToGateway(ctx, &MintToGatewayRequest{
		Amount: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(10),
		},
	}), "failed to mint LOOM: not authorized", "only gateway can call MintToGateway")

	require.EqualError(t, contract.MintToGateway(
		contractpb.WrapPluginContext(pctx.WithSender(binanceTGAddress)),
		&MintToGatewayRequest{
			Amount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(10),
			},
		}),
		"failed to mint LOOM: not authorized",
		"Binance gateway shouldn't be allowed to mint if coin:v1.3 is disabled",
	)

	pctx.SetFeature(features.CoinVersion1_3Feature, true)
	require.Nil(t, contract.MintToGateway(
		contractpb.WrapPluginContext(pctx.WithSender(binanceTGAddress)),
		&MintToGatewayRequest{
			Amount: &types.BigUInt{
				Value: *loom.NewBigUIntFromInt(10),
			},
		},
	), "Binance gateway should be allowed to call MintToGateway")
}

func TestNilRequest(t *testing.T) {
	pctx := plugin.CreateFakeContext(addr1, addr1)
	pctx.SetFeature(features.CoinVersion1_2Feature, true)
	ctx := contractpb.WrapPluginContext(pctx)
	contract := &Coin{}

	contract.Init(ctx, &InitRequest{
		Accounts: []*InitialAccount{
			{
				Owner:   addr1.MarshalPB(),
				Balance: 100,
			},
		},
	})
	err := contract.Burn(ctx, &BurnRequest{
		Owner:  nil,
		Amount: nil,
	})
	require.EqualError(t, err, "owner or amount is nil")

	balResp, err := contract.BalanceOf(ctx, &BalanceOfRequest{
		Owner: nil,
	})
	require.Equal(t, err, ErrInvalidRequest)
	require.Nil(t, balResp)

	err = contract.Transfer(ctx, &TransferRequest{
		To:     nil,
		Amount: nil,
	})
	require.Equal(t, err, ErrInvalidRequest)

	err = contract.Approve(ctx, &ApproveRequest{
		Spender: nil,
		Amount:  nil,
	})
	require.Equal(t, err, ErrInvalidRequest)

	amount := sciNot(0, 18)
	err = contract.Approve(ctx, &ApproveRequest{
		Spender: addr2.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *amount,
		},
	})
	require.NoError(t, err)

	alwResp, err := contract.Allowance(ctx, &AllowanceRequest{
		Owner:   nil,
		Spender: nil,
	})
	require.Equal(t, err, ErrInvalidRequest)
	require.Nil(t, alwResp)

	err = contract.TransferFrom(ctx, &TransferFromRequest{
		From:   addr2.MarshalPB(),
		To:     addr1.MarshalPB(),
		Amount: nil,
	})
	require.Equal(t, err, ErrInvalidRequest)

	err = contract.Transfer(ctx, &TransferRequest{
		To:     nil,
		Amount: nil,
	})
	assert.Equal(t, ErrInvalidRequest, err)

	err = contract.TransferFrom(ctx, &TransferFromRequest{
		From: addr2.MarshalPB(),
		To:   addr1.MarshalPB(),
		Amount: &types.BigUInt{
			Value: *amount,
		},
	})
	require.NoError(t, err)
}
