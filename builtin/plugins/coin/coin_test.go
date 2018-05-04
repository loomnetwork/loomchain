package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loom "github.com/loomnetwork/go-loom"
	cointypes "github.com/loomnetwork/go-loom/builtin/types/coin"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
)

var (
	addr1 = loom.MustParseAddress("chain:b16a379ec18d4093666f8f38b11a3071c920207d")
	addr2 = loom.MustParseAddress("chain:fa4c7920accfd66b86f5fd0e69682a79f762d49e")
)

func TestTransfer(t *testing.T) {
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr1, addr1),
	)

	amount := loom.NewBigUIntFromInt(100)
	contract := &Coin{}
	err := contract.Transfer(ctx, &cointypes.TransferRequest{
		To:     addr2.MarshalPB(),
		Amount: &types.BigUInt{Value: *amount},
	})
	assert.NotNil(t, err)

	acct := &cointypes.Account{
		Owner: addr1.MarshalPB(),
		Balance: &types.BigUInt{
			Value: *loom.NewBigUIntFromInt(100),
		},
	}
	err = saveAccount(ctx, acct)
	require.Nil(t, err)

	err = contract.Transfer(ctx, &cointypes.TransferRequest{
		To:     addr2.MarshalPB(),
		Amount: &types.BigUInt{Value: *amount},
	})
	assert.Nil(t, err)

	resp, err := contract.BalanceOf(ctx, &cointypes.BalanceOfRequest{
		Owner: addr1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, 0, int(resp.Balance.Value.Int64()))

	resp, err = contract.BalanceOf(ctx, &cointypes.BalanceOfRequest{
		Owner: addr2.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, 100, int(resp.Balance.Value.Int64()))
}
