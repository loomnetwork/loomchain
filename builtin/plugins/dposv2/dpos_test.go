package dposv2

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
	common "github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

var (
	valPubKeyHex1 = "3866f776276246e4f9998aa90632931d89b0d3a5930e804e02299533f55b39e1"
	valPubKeyHex2 = "7796b813617b283f81ea1747fbddbe73fe4b5fce0eac0728e47de51d8e506701"
	valPubKeyHex3 = "e4008e26428a9bca87465e8de3a8d0e9c37a56ca619d3d6202b0567528786618"

	voterAddr1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	voterAddr2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	voterAddr3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

func TestRegisterCandidate(t *testing.T) {
	c := &DPOS{}

	pubKey, _ := hex.DecodeString(valPubKeyHex1)
	addr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey),
	}
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr, addr),
	)

	err := c.RegisterCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey,
	})
	require.Nil(t, err)
}

func TestDelegate(t *testing.T) {
	c := &DPOS{}
	pubKey1, _ := hex.DecodeString(valPubKeyHex1)
	addr1 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey1),
	}
	pctx := plugin.CreateFakeContext(addr1, addr1)

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

	err = c.RegisterCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	/*
	// Too many votes given
	ctx = contractpb.WrapPluginContext(pctx.WithSender(voterAddr1))
	err = c.Delegate(ctx, &VoteRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           50,
	})
	require.NotNil(t, err)

	err = c.Vote(ctx, &VoteRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           20,
	})
	require.Nil(t, err)

	err = c.Vote(ctx, &VoteRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           2,
	})
	require.NotNil(t, err)
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
		DelegationTotal: &types.BigUInt{Value: delegationAmount},
	}
	for i := int64(0); i < yearSeconds; i = i + cycleLengthSeconds {
		rewardValidator(&statistic, &params)
	}

	// checking that distribution is roughtly equal to 7% of delegation after one year
	assert.Equal(t, statistic.DistributionTotal.Value.Cmp(&loom.BigUInt{big.NewInt(690000000000)}), 1)
	assert.Equal(t, statistic.DistributionTotal.Value.Cmp(&loom.BigUInt{big.NewInt(710000000000)}), -1)
}

func makeAccount(owner loom.Address, bal uint64) *coin.InitialAccount {
	return &coin.InitialAccount{
		Owner:   owner.MarshalPB(),
		Balance: bal,
	}
}

func TestElect(t *testing.T) {
	chainID := "chain"
	pubKey1, _ := hex.DecodeString(valPubKeyHex1)
	addr1 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey1),
	}

	pubKey2, _ := hex.DecodeString(valPubKeyHex2)
	addr2 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey2),
	}

	pubKey3, _ := hex.DecodeString(valPubKeyHex3)
	addr3 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey3),
	}

	// Init the coin balances
	var startTime int64 = 100000
	pctx := plugin.CreateFakeContext(voterAddr1, loom.Address{}).WithBlock(loom.BlockHeader{
		ChainID: chainID,
		Time:    startTime,
	})
	coinAddr := pctx.CreateContract(coin.Contract)

	coinContract := &coin.Coin{}
	ctx := contractpb.WrapPluginContext(pctx.WithAddress(coinAddr))
	coinContract.Init(ctx, &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(voterAddr1, 130),
			makeAccount(voterAddr2, 20),
			makeAccount(voterAddr3, 10),
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
	err = c.RegisterCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	ctx = contractpb.WrapPluginContext(pctx.WithSender(addr2))
	err = c.RegisterCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	ctx = contractpb.WrapPluginContext(pctx.WithSender(addr3))
	err = c.RegisterCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey3,
	})
	require.Nil(t, err)

	/*
	ctx = contractpb.WrapPluginContext(pctx.WithSender(voterAddr1))
	err = c.Vote(ctx, &VoteRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           10,
	})
	require.Nil(t, err)

	ctx = contractpb.WrapPluginContext(pctx.WithSender(voterAddr2))
	err = c.Vote(ctx, &VoteRequest{
		CandidateAddress: addr2.MarshalPB(),
		Amount:           12,
	})
	require.Nil(t, err)

	ctx = contractpb.WrapPluginContext(pctx.WithSender(voterAddr3))
	err = c.Vote(ctx, &VoteRequest{
		CandidateAddress: addr3.MarshalPB(),
		Amount:           20,
	})
	require.Nil(t, err)

	// Run the election
	ctx = contractpb.WrapPluginContext(pctx.WithBlock(loom.BlockHeader{
		ChainID: chainID,
		Time:    startTime + 3600,
	}))
	err = c.Elect(ctx, &ElectRequest{})
	require.Nil(t, err)

	resp, err := c.ListValidators(ctx, &ListValidatorsRequest{})
	require.Nil(t, err)
	validators := resp.Validators
	require.Len(t, validators, 2)
	assert.Equal(t, pubKey1, validators[0].PubKey)
	assert.Equal(t, 300, int(validators[0].Power))
	assert.Equal(t, pubKey2, validators[1].PubKey)
	assert.Equal(t, 240, int(validators[1].Power))

	valids := pctx.Validators()
	require.Len(t, valids, 2)
	assert.Equal(t, pubKey1, valids[0].PubKey)
	assert.Equal(t, pubKey2, valids[1].PubKey)

	// Shouldn't be able to elect again for an hour
	ctx = contractpb.WrapPluginContext(pctx.WithBlock(loom.BlockHeader{
		ChainID: chainID,
		Time:    startTime + 6000,
	}))
	err = c.Elect(ctx, &ElectRequest{})
	require.NotNil(t, err)

	// Waited an hour, should be able to elect again
	ctx = contractpb.WrapPluginContext(pctx.WithBlock(loom.BlockHeader{
		ChainID: chainID,
		Time:    startTime + 7300,
	}))

	// Run the election again. should get same results as no votes changed
	err = c.Elect(ctx, &ElectRequest{})
	require.Nil(t, err)

	resp, err = c.ListValidators(ctx, &ListValidatorsRequest{})
	require.Nil(t, err)
	validators = resp.Validators
	require.Len(t, validators, 2)
	assert.Equal(t, pubKey1, validators[0].PubKey)
	assert.Equal(t, 300, int(validators[0].Power))
	assert.Equal(t, pubKey2, validators[1].PubKey)
	assert.Equal(t, 240, int(validators[1].Power))

	valids = pctx.Validators()
	require.Len(t, valids, 2)
	assert.Equal(t, pubKey1, valids[0].PubKey)
	assert.Equal(t, pubKey2, valids[1].PubKey)

	staticCoin := &ERC20Static{
		StaticContext:   ctx,
		ContractAddress: coinAddr,
	}

	// check the reward balances
	for _, wit := range validators {
		witLocalAddr := loom.LocalAddressFromPublicKey(wit.PubKey)
		witAddr := loom.Address{ChainID: chainID, Local: witLocalAddr}
		bal, err := staticCoin.BalanceOf(witAddr)
		assert.Nil(t, err)
		assert.Equal(t, sciNot(10, 18), bal)
	}
	*/
}
