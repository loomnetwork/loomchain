package ethcoin

import (
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

func TestTransfer(t *testing.T) {
	pctx := plugin.CreateFakeContext(addr1, addr2)
	ctx := contractpb.WrapPluginContext(pctx)

	amount := loom.NewBigUIntFromInt(100)
	contract := &ETHCoin{}
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
	resp, err = contract.BalanceOf(contractpb.WrapPluginStaticContext(pctx), &BalanceOfRequest{
		Owner: nil,
	})
	require.Error(t, err, ErrInvalidRequest)
	require.Nil(t, resp)

}

func sciNot(m, n int64) *loom.BigUInt {
	ret := loom.NewBigUIntFromInt(10)
	ret.Exp(ret, loom.NewBigUIntFromInt(n), nil)
	ret.Mul(ret, loom.NewBigUIntFromInt(m))
	return ret
}

// Verify ETHCoin.Transfer works correctly when the to & from addresses are the same.
func TestTransferToSelf(t *testing.T) {
	pctx := plugin.CreateFakeContext(addr1, addr1)
	// Test using the v1.1 contract, this test will fail if this feature is not enabled
	pctx.SetFeature(features.CoinVersion1_1Feature, true)

	contract := &ETHCoin{}
	amount := sciNot(100, 18)
	require.NoError(t, Mint(contractpb.WrapPluginContext(pctx), addr2, amount))

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

	pctx.SetFeature(features.CoinVersion1_2Feature, true)
	err = contract.Transfer(contractpb.WrapPluginContext(pctx), &TransferRequest{
		To:     nil,
		Amount: &types.BigUInt{Value: *amount},
	})
	require.Error(t, err)
	assert.Equal(t, ErrInvalidRequest, err)
}

func TestApprove(t *testing.T) {
	contract := &ETHCoin{}

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

	pctx := plugin.CreateFakeContext(addr1, addr2)
	pctx.SetFeature(features.CoinVersion1_2Feature, true)

	err = contract.Approve(contractpb.WrapPluginContext(pctx), &ApproveRequest{
		Spender: nil,
		Amount:  nil,
	})
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidRequest, err)

	resp, err := contract.Allowance(contractpb.WrapPluginContext(pctx), &AllowanceRequest{
		Owner:   nil,
		Spender: nil,
	})
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidRequest, err)
	require.Nil(t, resp)

}

func TestTransferFrom(t *testing.T) {
	contract := &ETHCoin{}

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
	amount := sciNot(100, 18)
	err = contract.TransferFrom(contractpb.WrapPluginContext(pctx), &TransferFromRequest{
		To:     nil,
		From:   nil,
		Amount: &types.BigUInt{Value: *amount},
	})
	require.Error(t, err)
	assert.Equal(t, ErrInvalidRequest, err)
}

// Verify ETHCoin.TransferFrom works correctly when the to & from addresses are the same.
func TestTransferFromSelf(t *testing.T) {
	pctx := plugin.CreateFakeContext(addr1, addr1)
	// Test using the v1.1 contract, this test will fail if this feature is not enabled
	pctx.SetFeature(features.CoinVersion1_1Feature, true)

	contract := &ETHCoin{}
	amount := sciNot(100, 18)
	require.NoError(t, Mint(contractpb.WrapPluginContext(pctx), addr2, amount))

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
