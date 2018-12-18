package dposv2

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loom "github.com/loomnetwork/go-loom"
	common "github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

var (
	validatorPubKeyHex1 = "3866f776276246e4f9998aa90632931d89b0d3a5930e804e02299533f55b39e1"
	validatorPubKeyHex2 = "7796b813617b283f81ea1747fbddbe73fe4b5fce0eac0728e47de51d8e506701"
	validatorPubKeyHex3 = "e4008e26428a9bca87465e8de3a8d0e9c37a56ca619d3d6202b0567528786618"

	delegatorAddress1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	delegatorAddress2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	delegatorAddress3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestRegisterWhitelistedCandidate(t *testing.T) {
	c := &DPOS{}

	pubKey, _ := hex.DecodeString(validatorPubKeyHex1)
	addr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey),
	}
	pctx := plugin.CreateFakeContext(addr, addr)

	// Deploy the coin contract (DPOS Init() will attempt to resolve it)
	coinContract := &coin.Coin{}
	_ = pctx.CreateContract(contractpb.MakePluginContract(coinContract))

	ctx := contractpb.WrapPluginContext(pctx)
	err := c.Init(ctx, &InitRequest{
		Params: &Params{
			ValidatorCount: 21,
		},
	})
	require.Nil(t, err)

	err = c.WhitelistCandidate(ctx, &WhitelistCandidateRequest{
			CandidateAddress: addr.MarshalPB(),
			Amount: &types.BigUInt{Value: loom.BigUInt{big.NewInt(1000000000000)}},
			LockTime: 10,
	})
	require.Nil(t, err)

	err = c.RegisterCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey,
	})
	require.Nil(t, err)
}

func TestDelegate(t *testing.T) {
	c := &DPOS{}
	pubKey1, _ := hex.DecodeString(validatorPubKeyHex1)
	addr1 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey1),
	}
	pctx := plugin.CreateFakeContext(addr1, addr1)

	// Deploy the coin contract (DPOS Init() will attempt to resolve it)
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	ctx := contractpb.WrapPluginContext(pctx.WithAddress(coinAddr))
	coinContract.Init(ctx, &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 1000000000000000000),
			makeAccount(delegatorAddress2, 2000000000000000000),
			makeAccount(delegatorAddress3, 1000000000000000000),
		},
	})

	_ = pctx.CreateContract(contractpb.MakePluginContract(coinContract))

	err := c.Init(ctx, &InitRequest{
		Params: &Params{
			ValidatorCount: 21,
		},
	})
	require.Nil(t, err)

	err = c.registerCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	// Delegate to this candidate
	dposAddr := pctx.CreateContract(Contract)
	delegationAmount := &types.BigUInt{Value: loom.BigUInt{big.NewInt(100)}}
	err = coinContract.Approve(ctx, &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount: delegationAmount,
	})
	require.Nil(t, err)

	response, err := coinContract.Allowance(ctx, &coin.AllowanceRequest{
		Owner: addr1.MarshalPB(),
		Spender: dposAddr.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegationAmount.Value.Int64(), response.Amount.Value.Int64())

	/*
	err = c.Delegate(ctx, &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount: delegationAmount,
	})
	require.Nil(t, err)
	*/
}

func TestReward(t *testing.T) {
	// set elect time in params to one second for easy calculations
	delegationAmount := loom.BigUInt{big.NewInt(10000000000000)}
	cycleLengthSeconds := int64(100)
	params := Params{
		ElectionCycleLength: cycleLengthSeconds,
	}
	statistic := ValidatorStatistic{
		DistributionTotal: &types.BigUInt{Value: loom.BigUInt{big.NewInt(0)}},
		DelegationTotal:   &types.BigUInt{Value: delegationAmount},
	}
	for i := int64(0); i < yearSeconds; i = i + cycleLengthSeconds {
		rewardValidator(&statistic, &params)
	}

	// checking that distribution is roughtly equal to 7% of delegation after one year
	assert.Equal(t, statistic.DistributionTotal.Value.Cmp(&loom.BigUInt{big.NewInt(690000000000)}), 1)
	assert.Equal(t, statistic.DistributionTotal.Value.Cmp(&loom.BigUInt{big.NewInt(710000000000)}), -1)
}

func TestElect(t *testing.T) {
	chainID := "chain"
	pubKey1, _ := hex.DecodeString(validatorPubKeyHex1)
	addr1 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey1),
	}

	pubKey2, _ := hex.DecodeString(validatorPubKeyHex2)
	addr2 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey2),
	}

	pubKey3, _ := hex.DecodeString(validatorPubKeyHex3)
	addr3 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey3),
	}

	// Init the coin balances
	var startTime int64 = 100000
	pctx := plugin.CreateFakeContext(delegatorAddress1, loom.Address{}).WithBlock(loom.BlockHeader{
		ChainID: chainID,
		Time:    startTime,
	})
	coinAddr := pctx.CreateContract(coin.Contract)

	coinContract := &coin.Coin{}
	ctx := contractpb.WrapPluginContext(pctx.WithAddress(coinAddr))
	coinContract.Init(ctx, &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 130),
			makeAccount(delegatorAddress2, 20),
			makeAccount(delegatorAddress3, 10),
		},
	})

	// create dpos contract
	c := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(c))

	// transfer coins to reward fund from voter1
	// TODO clean this up, something better than the sciNot impl of before
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(19), nil)
	coinContract.Transfer(ctx, &coin.TransferRequest{
		To: dposAddr.MarshalPB(),
		Amount: &types.BigUInt{
			Value: common.BigUInt{amount},
		},
	})

	// Switch to dpos contract context
	pctx = pctx.WithAddress(dposAddr)

	// Init the dpos contract
	ctx = contractpb.WrapPluginContext(pctx.WithSender(addr1))
	err := c.Init(ctx, &InitRequest{
		Params: &Params{
			CoinContractAddress: coinAddr.MarshalPB(),
			ValidatorCount:      2,
			ElectionCycleLength: 3600,
		},
	})
	require.Nil(t, err)

	// Register candidates
	ctx = contractpb.WrapPluginContext(pctx.WithSender(addr1))
	err = c.registerCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	ctx = contractpb.WrapPluginContext(pctx.WithSender(addr2))
	err = c.registerCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	ctx = contractpb.WrapPluginContext(pctx.WithSender(addr3))
	err = c.registerCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey3,
	})
	require.Nil(t, err)
}

// UTILITIES

func makeAccount(owner loom.Address, bal uint64) *coin.InitialAccount {
	return &coin.InitialAccount{
		Owner:   owner.MarshalPB(),
		Balance: bal,
	}
}
