package dposv3

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
	validatorPubKeyHex4 = "21908210428a9bca87465e8de3a8d0e9c37a56ca619d3d6202b0567528786618"

	delegatorAddress1       = loom.MustParseAddress("default:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	delegatorAddress2       = loom.MustParseAddress("default:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	delegatorAddress3       = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	delegatorAddress4       = loom.MustParseAddress("default:0x000000000000000000000000e3edf03b825e01e0")
	delegatorAddress5       = loom.MustParseAddress("default:0x020000000000000000000000e3edf03b825e0288")
	delegatorAddress6       = loom.MustParseAddress("default:0x000000000000000000040400e3edf03b825e0398")
	chainID                 = "default"
	startTime         int64 = 100000

	pubKey1, _ = hex.DecodeString(validatorPubKeyHex1)
	addr1      = loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey1),
	}
	pubKey2, _ = hex.DecodeString(validatorPubKeyHex2)
	addr2      = loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey2),
	}
	pubKey3, _ = hex.DecodeString(validatorPubKeyHex3)
	addr3      = loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey3),
	}
	pubKey4, _ = hex.DecodeString(validatorPubKeyHex4)
	addr4      = loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey4),
	}
)

func TestElect(t *testing.T) {
	pctx := createCtx()
	coinAddr := pctx.CreateContract(coin.Contract)

	coinContract := &coin.Coin{}
	coinCtx := pctx.WithAddress(coinAddr)
	// Initialize the coin balances
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 130),
			makeAccount(delegatorAddress2, 20),
			makeAccount(delegatorAddress3, 10),
		},
	})

	// create dpos contract
	dpos, err := deployDPOSContract(pctx, &Params{
		ValidatorCount:      2,
		CoinContractAddress: coinAddr.MarshalPB(),
		OracleAddress:       addr1.MarshalPB(),
	})
	require.Nil(t, err)

	// transfer coins to reward fund
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(19), nil)
	coinContract.Transfer(contractpb.WrapPluginContext(coinCtx), &coin.TransferRequest{
		To: dpos.Address.MarshalPB(),
		Amount: &types.BigUInt{
			Value: common.BigUInt{amount},
		},
	})

	registrationFee := &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}
	smallDelegationAmount := loom.NewBigUIntFromInt(0)
	smallDelegationAmount.Div(&registrationFee.Value, loom.NewBigUIntFromInt(4))

	whitelistAmount := big.NewInt(1000000000000)

	err = dpos.WhitelistCandidate(pctx.WithSender(addr1), addr1, whitelistAmount, 0)
	require.Nil(t, err)

	err = dpos.WhitelistCandidate(pctx.WithSender(addr1), addr2, whitelistAmount, 0)
	require.Nil(t, err)

	err = dpos.WhitelistCandidate(pctx.WithSender(addr1), addr3, whitelistAmount, 0)
	require.Nil(t, err)

	err = dpos.RegisterCandidate(pctx.WithSender(addr1), pubKey1, nil, nil, nil, nil, nil, nil)
	require.Nil(t, err)

	err = dpos.RegisterCandidate(pctx.WithSender(addr2), pubKey2, nil, nil, nil, nil, nil, nil)
	require.Nil(t, err)

	err = dpos.RegisterCandidate(pctx.WithSender(addr3), pubKey3, nil, nil, nil, nil, nil, nil)
	require.Nil(t, err)

	candidates, err := dpos.ListCandidates(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(candidates), 3)

	validators, err := dpos.ListValidators(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(validators), 0)

	err = dpos.Delegate(pctx.WithSender(delegatorAddress1), &addr1, smallDelegationAmount.Int, nil, nil)
	require.Nil(t, err)

	require.NoError(t, elect(pctx, dpos.Address))

	validators, err = dpos.ListValidators(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(validators), 2)

	oldRewardsValue := big.NewInt(0)
	for i := 0; i < 10; i++ {
		require.NoError(t, elect(pctx, dpos.Address))
		delegations, amount, _, err := dpos.CheckDelegation(pctx.WithSender(addr1), &addr1, &addr1)
		require.NoError(t, err)
		// get rewards delegaiton which is always at index 0
		delegation := delegations[REWARD_DELEGATION_INDEX]
		assert.True(t, delegation.Amount.Value.Int.Cmp(oldRewardsValue) == 1)
		oldRewardsValue = amount
	}

	// Change WhitelistAmount and verify that it got changed correctly
	validators, err = dpos.ListValidators(pctx)
	require.Nil(t, err)
	validator := validators[0]
	assert.Equal(t, whitelistAmount, validator.WhitelistAmount.Value.Int)

	newWhitelistAmount := big.NewInt(2000000000000)
	newTier := TIER_THREE

	// only oracle
	err = dpos.ChangeWhitelistInfo(pctx.WithSender(addr2), &addr1, newWhitelistAmount, nil)
	require.Equal(t, errOnlyOracle, err)

	err = dpos.ChangeWhitelistInfo(pctx.WithSender(addr1), &addr1, newWhitelistAmount, &newTier)
	require.Nil(t, err)

	require.NoError(t, elect(pctx, dpos.Address))

	validators, err = dpos.ListValidators(pctx)
	require.Nil(t, err)
	validator = validators[0]
	assert.Equal(t, newWhitelistAmount, validator.WhitelistAmount.Value.Int)
	assert.Equal(t, newTier, validator.LocktimeTier)
}

func TestValidatorRewards(t *testing.T) {
	// Init the coin balances
	pctx := createCtx()
	coinAddr := pctx.CreateContract(coin.Contract)

	coinContract := &coin.Coin{}
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 100000000),
			makeAccount(delegatorAddress2, 100000000),
			makeAccount(delegatorAddress3, 100000000),
			makeAccount(addr1, 100000000),
			makeAccount(addr2, 100000000),
			makeAccount(addr3, 100000000),
		},
	})

	// create dpos contract
	dpos, err := deployDPOSContract(pctx, &Params{
		ValidatorCount:      10,
		CoinContractAddress: coinAddr.MarshalPB(),
	})
	require.Nil(t, err)

	// transfer coins to reward fund
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(19), nil)
	coinContract.Transfer(contractpb.WrapPluginContext(coinCtx), &coin.TransferRequest{
		To: dpos.Address.MarshalPB(),
		Amount: &types.BigUInt{
			Value: common.BigUInt{amount},
		},
	})

	registrationFee := &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dpos.RegisterCandidate(pctx.WithSender(addr1), pubKey1, nil, nil, nil, nil, nil, nil)
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr2)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dpos.RegisterCandidate(pctx.WithSender(addr2), pubKey2, nil, nil, nil, nil, nil, nil)
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr3)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dpos.RegisterCandidate(pctx.WithSender(addr3), pubKey3, nil, nil, nil, nil, nil, nil)
	require.Nil(t, err)

	candidates, err := dpos.ListCandidates(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(candidates), 3)

	validators, err := dpos.ListValidators(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(validators), 0)

	require.NoError(t, elect(pctx, dpos.Address))

	validators, err = dpos.ListValidators(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(validators), 3)

	// Two delegators delegate 1/2 and 1/4 of a registration fee respectively
	smallDelegationAmount := loom.NewBigUIntFromInt(0)
	smallDelegationAmount.Div(&registrationFee.Value, loom.NewBigUIntFromInt(4))
	largeDelegationAmount := loom.NewBigUIntFromInt(0)
	largeDelegationAmount.Div(&registrationFee.Value, loom.NewBigUIntFromInt(2))

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	err = dpos.Delegate(pctx.WithSender(delegatorAddress1), &addr1, smallDelegationAmount.Int, nil, nil)
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress2)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  &types.BigUInt{Value: *largeDelegationAmount},
	})
	require.Nil(t, err)

	err = dpos.Delegate(pctx.WithSender(delegatorAddress2), &addr1, largeDelegationAmount.Int, nil, nil)
	require.Nil(t, err)

	for i := 0; i < 10000; i++ {
		require.NoError(t, elect(pctx, dpos.Address))
	}

	_, amount, _, err = dpos.CheckDelegation(pctx.WithSender(addr1), &addr1, &addr1)
	require.Nil(t, err)
	assert.Equal(t, amount.Cmp(big.NewInt(0)), 1)

	_, delegator1Claim, _, err := dpos.CheckDelegation(pctx.WithSender(delegatorAddress1), &addr1, &delegatorAddress1)
	require.Nil(t, err)
	assert.Equal(t, delegator1Claim.Cmp(big.NewInt(0)), 1)

	_, delegator2Claim, _, err := dpos.CheckDelegation(pctx.WithSender(delegatorAddress2), &addr1, &delegatorAddress2)
	require.Nil(t, err)
	assert.Equal(t, delegator2Claim.Cmp(big.NewInt(0)), 1)

	halvedDelegator2Claim := big.NewInt(0)
	halvedDelegator2Claim.Div(delegator2Claim, big.NewInt(2))
	difference := big.NewInt(0)
	difference.Sub(delegator1Claim, halvedDelegator2Claim)

	// Checking that Delegator2's claim is almost exactly half of Delegator1's claim
	maximumDifference := scientificNotation(1, tokenDecimals)
	assert.Equal(t, difference.CmpAbs(maximumDifference.Int), -1)

	// Using unbond to claim reward delegation
	err = dpos.Unbond(pctx.WithSender(addr1), &addr1, big.NewInt(0), REWARD_DELEGATION_INDEX)
	require.Nil(t, err)

	// check that addr1's balance increases after rewards claim
	balanceBeforeUnbond, err := coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx), &coin.BalanceOfRequest{
		Owner: addr1.MarshalPB(),
	})
	require.Nil(t, err)

	// allowing reward delegation to unbond
	require.NoError(t, elect(pctx, dpos.Address))
	require.Nil(t, err)

	balanceAfterUnbond, err := coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx), &coin.BalanceOfRequest{
		Owner: addr1.MarshalPB(),
	})
	require.Nil(t, err)

	assert.True(t, balanceAfterUnbond.Balance.Value.Cmp(&balanceBeforeUnbond.Balance.Value) > 0)

	// check that difference is exactly the undelegated amount

	// check current delegation amount
}

func TestReferrerRewards(t *testing.T) {
	// Init the coin balances
	pctx := createCtx()
	limboValidatorAddress := LimboValidatorAddress(contractpb.WrapPluginStaticContext(pctx))
	coinAddr := pctx.CreateContract(coin.Contract)

	coinContract := &coin.Coin{}
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 100000000),
			makeAccount(delegatorAddress2, 100000000),
			makeAccount(delegatorAddress3, 100000000),
			makeAccount(addr1, 100000000),
		},
	})

	// create dpos contract
	dpos, err := deployDPOSContract(pctx, &Params{
		ValidatorCount:      10,
		CoinContractAddress: coinAddr.MarshalPB(),
		OracleAddress:       addr1.MarshalPB(),
	})
	require.Nil(t, err)

	registrationFee := &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	fee := uint64(2000)
	pct := uint64(10000)
	err = dpos.RegisterCandidate(pctx.WithSender(addr1), pubKey1, nil, &fee, &pct, nil, nil, nil)
	require.Nil(t, err)

	candidates, err := dpos.ListCandidates(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(candidates), 1)

	validators, err := dpos.ListValidators(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(validators), 0)

	require.NoError(t, elect(pctx, dpos.Address))

	validators, err = dpos.ListValidators(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(validators), 1)

	del1Name := "del1"
	// Register two referrers
	err = dpos.RegisterReferrer(pctx.WithSender(addr1), delegatorAddress1, "del1")
	require.Nil(t, err)

	err = dpos.RegisterReferrer(pctx.WithSender(addr1), delegatorAddress2, "del2")
	require.Nil(t, err)

	delegationAmount := big.NewInt(1e18)
	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress3)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  &types.BigUInt{Value: *loom.NewBigUInt(delegationAmount)},
	})
	require.Nil(t, err)

	err = dpos.Delegate(pctx.WithSender(delegatorAddress3), &addr1, delegationAmount, nil, &del1Name)
	require.Nil(t, err)

	for i := 0; i < 10; i++ {
		require.NoError(t, elect(pctx, dpos.Address))
	}

	_, amount, _, err := dpos.CheckDelegation(pctx.WithSender(addr1), &limboValidatorAddress, &delegatorAddress1)
	require.Nil(t, err)
	assert.Equal(t, amount.Cmp(big.NewInt(0)), 1)
}

func TestRewardTiers(t *testing.T) {
	// Init the coin balances
	pctx := createCtx()
	coinAddr := pctx.CreateContract(coin.Contract)

	coinContract := &coin.Coin{}
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 100000000),
			makeAccount(delegatorAddress2, 100000000),
			makeAccount(delegatorAddress3, 100000000),
			makeAccount(delegatorAddress4, 100000000),
			makeAccount(delegatorAddress5, 100000000),
			makeAccount(delegatorAddress6, 100000000),
			makeAccount(addr1, 100000000),
			makeAccount(addr2, 100000000),
			makeAccount(addr3, 100000000),
		},
	})

	// Init the dpos contract
	dpos, err := deployDPOSContract(pctx, &Params{
		ValidatorCount:      10,
		CoinContractAddress: coinAddr.MarshalPB(),
	})
	require.Nil(t, err)

	// transfer coins to reward fund
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(19), nil)
	coinContract.Transfer(contractpb.WrapPluginContext(coinCtx), &coin.TransferRequest{
		To: dpos.Address.MarshalPB(),
		Amount: &types.BigUInt{
			Value: common.BigUInt{amount},
		},
	})

	registrationFee := &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dpos.RegisterCandidate(pctx.WithSender(addr1), pubKey1, nil, nil, nil, nil, nil, nil)
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr2)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dpos.RegisterCandidate(pctx.WithSender(addr2), pubKey2, nil, nil, nil, nil, nil, nil)
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr3)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dpos.RegisterCandidate(pctx.WithSender(addr3), pubKey3, nil, nil, nil, nil, nil, nil)
	require.Nil(t, err)

	candidates, err := dpos.ListCandidates(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(candidates), 3)

	validators, err := dpos.ListValidators(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(validators), 0)

	require.NoError(t, elect(pctx, dpos.Address))

	validators, err = dpos.ListValidators(pctx)
	require.Nil(t, err)
	assert.Equal(t, len(validators), 3)

	// tinyDelegationAmount = one LOOM token
	tinyDelegationAmount := scientificNotation(1, tokenDecimals)
	smallDelegationAmount := loom.NewBigUIntFromInt(0)
	smallDelegationAmount.Div(&registrationFee.Value, loom.NewBigUIntFromInt(4))
	largeDelegationAmount := loom.NewBigUIntFromInt(0)
	largeDelegationAmount.Div(&registrationFee.Value, loom.NewBigUIntFromInt(2))

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	// LocktimeTier should default to 0 for delegatorAddress1
	err = dpos.Delegate(pctx.WithSender(delegatorAddress1), &addr1, smallDelegationAmount.Int, nil, nil)
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress2)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	tier := uint64(2)
	err = dpos.Delegate(pctx.WithSender(delegatorAddress2), &addr1, smallDelegationAmount.Int, &tier, nil)
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress3)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	tier = uint64(3)
	err = dpos.Delegate(pctx.WithSender(delegatorAddress3), &addr1, smallDelegationAmount.Int, &tier, nil)
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress4)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	tier = uint64(1)
	err = dpos.Delegate(pctx.WithSender(delegatorAddress4), &addr1, smallDelegationAmount.Int, &tier, nil)
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress5)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  &types.BigUInt{Value: *largeDelegationAmount},
	})
	require.Nil(t, err)

	// Though Delegator5 delegates to addr2 and not addr1 like the rest of the
	// delegators, he should still receive the same rewards proportional to his
	// delegation parameters
	tier = uint64(2)
	err = dpos.Delegate(pctx.WithSender(delegatorAddress5), &addr2, largeDelegationAmount.Int, &tier, nil)
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress6)), &coin.ApproveRequest{
		Spender: dpos.Address.MarshalPB(),
		Amount:  &types.BigUInt{Value: *tinyDelegationAmount},
	})
	require.Nil(t, err)

	// by delegating a very small amount, delegator6 demonstrates that
	// delegators can contribute far less than 0.01% of a validator's total
	// delegation and still be rewarded
	err = dpos.Delegate(pctx.WithSender(delegatorAddress6), &addr1, tinyDelegationAmount.Int, nil, nil)
	require.Nil(t, err)

	for i := 0; i < 10000; i++ {
		require.NoError(t, elect(pctx, dpos.Address))
	}

	addr1Claim, err := dpos.CheckRewardDelegation(pctx.WithSender(addr1), &addr1)
	require.Nil(t, err)
	assert.Equal(t, addr1Claim.Amount.Value.Cmp(common.BigZero()), 1)

	delegator1Claim, err := dpos.CheckRewardDelegation(pctx.WithSender(delegatorAddress1), &addr1)
	require.Nil(t, err)
	assert.Equal(t, delegator1Claim.Amount.Value.Cmp(common.BigZero()), 1)

	delegator2Claim, err := dpos.CheckRewardDelegation(pctx.WithSender(delegatorAddress2), &addr1)
	require.Nil(t, err)
	assert.Equal(t, delegator2Claim.Amount.Value.Cmp(common.BigZero()), 1)

	delegator3Claim, err := dpos.CheckRewardDelegation(pctx.WithSender(delegatorAddress3), &addr1)
	require.Nil(t, err)
	assert.Equal(t, delegator3Claim.Amount.Value.Cmp(common.BigZero()), 1)

	delegator4Claim, err := dpos.CheckRewardDelegation(pctx.WithSender(delegatorAddress4), &addr1)
	require.Nil(t, err)
	assert.Equal(t, delegator4Claim.Amount.Value.Cmp(common.BigZero()), 1)

	delegator5Claim, err := dpos.CheckRewardDelegation(pctx.WithSender(delegatorAddress5), &addr2)
	require.Nil(t, err)
	assert.Equal(t, delegator5Claim.Amount.Value.Cmp(common.BigZero()), 1)

	delegator6Claim, err := dpos.CheckRewardDelegation(pctx.WithSender(delegatorAddress6), &addr1)
	require.Nil(t, err)
	assert.Equal(t, delegator6Claim.Amount.Value.Cmp(common.BigZero()), 1)

	maximumDifference := scientificNotation(1, tokenDecimals)
	difference := loom.NewBigUIntFromInt(0)

	// Checking that Delegator2's claim is almost exactly twice Delegator1's claim
	scaledDelegator1Claim := CalculateFraction(*loom.NewBigUIntFromInt(20000), delegator1Claim.Amount.Value)
	difference.Sub(&scaledDelegator1Claim, &delegator2Claim.Amount.Value)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)

	// Checking that Delegator3's & Delegator5's claim is almost exactly four times Delegator1's claim
	scaledDelegator1Claim = CalculateFraction(*loom.NewBigUIntFromInt(40000), delegator1Claim.Amount.Value)

	difference.Sub(&scaledDelegator1Claim, &delegator3Claim.Amount.Value)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)

	difference.Sub(&scaledDelegator1Claim, &delegator5Claim.Amount.Value)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)

	// Checking that Delegator4's claim is almost exactly 1.5 times Delegator1's claim
	scaledDelegator1Claim = CalculateFraction(*loom.NewBigUIntFromInt(15000), delegator1Claim.Amount.Value)
	difference.Sub(&scaledDelegator1Claim, &delegator4Claim.Amount.Value)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)

	// Testing total delegation functionality

	_, amount, weightedAmount, err := dpos.CheckAllDelegations(pctx, &delegatorAddress3)
	require.Nil(t, err)
	assert.True(t, amount.Cmp(smallDelegationAmount.Int) > 0)
	expectedWeightedAmount := CalculateFraction(*loom.NewBigUIntFromInt(40000), *smallDelegationAmount)
	assert.True(t, weightedAmount.Cmp(expectedWeightedAmount.Int) > 0)
}

// UTILITIES

func makeAccount(owner loom.Address, bal uint64) *coin.InitialAccount {
	return &coin.InitialAccount{
		Owner:   owner.MarshalPB(),
		Balance: bal,
	}
}

func elect(pctx *plugin.FakeContext, dposAddress loom.Address) error {
	return Elect(contractpb.WrapPluginContext(pctx.WithAddress(dposAddress)))
}

func createCtx() *plugin.FakeContext {
	return plugin.CreateFakeContext(loom.Address{}, loom.Address{}).WithBlock(loom.BlockHeader{
		ChainID: chainID,
		Time:    startTime,
	})
}
