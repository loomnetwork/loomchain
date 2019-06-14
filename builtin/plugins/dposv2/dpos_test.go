package dposv2

import (
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loom "github.com/loomnetwork/go-loom"
	common "github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"

	d2types "github.com/loomnetwork/go-loom/builtin/types/dposv2"
)

var (
	validatorPubKeyHex1 = "3866f776276246e4f9998aa90632931d89b0d3a5930e804e02299533f55b39e1"
	validatorPubKeyHex2 = "7796b813617b283f81ea1747fbddbe73fe4b5fce0eac0728e47de51d8e506701"
	validatorPubKeyHex3 = "e4008e26428a9bca87465e8de3a8d0e9c37a56ca619d3d6202b0567528786618"
	validatorPubKeyHex4 = "7796b813617b283f81ea1747fbddbe73fe12347891249878e47de51d8e506701"
	validatorPubKeyHex5 = "e4008e120897323017495e8de3a8d0e9c37a56ca619d3d6202b0567528786618"

	delegatorAddress1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	delegatorAddress2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	delegatorAddress3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	delegatorAddress4 = loom.MustParseAddress("chain:0x000000000000000000000000e3edf03b825e01e0")
	delegatorAddress5 = loom.MustParseAddress("chain:0x020000000000000000000000e3edf03b825e0288")
	delegatorAddress6 = loom.MustParseAddress("chain:0x000000000000000000040400e3edf03b825e0398")
)

func TestChangeParams(t *testing.T) {
	oraclePubKey, _ := hex.DecodeString(validatorPubKeyHex2)
	oracleAddr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(oraclePubKey),
	}

	pubKey, _ := hex.DecodeString(validatorPubKeyHex1)
	addr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey),
	}

	pubKey2, _ := hex.DecodeString(validatorPubKeyHex3)
	addr2 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey2),
	}

	pctx := plugin.CreateFakeContext(addr, addr)

	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(addr2, 2000000000000000000),
		},
	})
	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)
	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &InitRequest{
		Params: &Params{
			ValidatorCount: 21,
			OracleAddress:  oracleAddr.MarshalPB(),
		},
	})
	require.NoError(t, err)

	// buggy set function

	// fails because not oracle
	err = dposContract.SetValidatorCount(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &SetValidatorCountRequest{
		ValidatorCount: 3,
	})
	require.Equal(t, errOnlyOracle, err)

	err = dposContract.SetValidatorCount(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &SetValidatorCountRequest{
		ValidatorCount: 3,
	})
	require.NoError(t, err)

	stateResponse, err := dposContract.GetState(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &GetStateRequest{})
	assert.Equal(t, stateResponse.State.Params.ValidatorCount, uint64(21))

	// fixed set validator count function

	// fails because not oracle
	err = dposContract.SetValidatorCount2(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &SetValidatorCountRequest{
		ValidatorCount: 3,
	})
	require.Equal(t, errOnlyOracle, err)

	err = dposContract.SetValidatorCount2(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &SetValidatorCountRequest{
		ValidatorCount: 3,
	})
	require.NoError(t, err)

	stateResponse, err = dposContract.GetState(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &GetStateRequest{})
	assert.Equal(t, stateResponse.State.Params.ValidatorCount, uint64(3))
	assert.Equal(t, stateResponse.State.Params.CrashSlashingPercentage.Value.Int64(), int64(100))
	assert.Equal(t, stateResponse.State.Params.ByzantineSlashingPercentage.Value.Int64(), int64(500))

	// set slashing percentages

	// fails because not oracle
	err = dposContract.SetSlashingPercentages(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &SetSlashingPercentagesRequest{
		CrashSlashingPercentage:     &types.BigUInt{Value: *loom.NewBigUIntFromInt(200)},
		ByzantineSlashingPercentage: &types.BigUInt{Value: *loom.NewBigUIntFromInt(50)},
	})
	require.Equal(t, errOnlyOracle, err)

	err = dposContract.SetSlashingPercentages(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &SetSlashingPercentagesRequest{
		CrashSlashingPercentage:     &types.BigUInt{Value: *loom.NewBigUIntFromInt(200)},
		ByzantineSlashingPercentage: &types.BigUInt{Value: *loom.NewBigUIntFromInt(50)},
	})
	require.NoError(t, err)

	stateResponse, err = dposContract.GetState(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &GetStateRequest{})
	assert.Equal(t, stateResponse.State.Params.CrashSlashingPercentage.Value.Int64(), int64(200))
	assert.Equal(t, stateResponse.State.Params.ByzantineSlashingPercentage.Value.Int64(), int64(50))

	// set registration requirement

	// fails because not oracle
	err = dposContract.SetRegistrationRequirement(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &SetRegistrationRequirementRequest{
		RegistrationRequirement: &types.BigUInt{Value: *loom.NewBigUIntFromInt(100)},
	})
	require.Equal(t, errOnlyOracle, err)

	err = dposContract.SetRegistrationRequirement(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &SetRegistrationRequirementRequest{
		RegistrationRequirement: &types.BigUInt{Value: *loom.NewBigUIntFromInt(100)},
	})
	require.NoError(t, err)

	stateResponse, err = dposContract.GetState(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &GetStateRequest{})
	assert.Equal(t, stateResponse.State.Params.RegistrationRequirement.Value.Int64(), int64(100))

	// set max yearly reward

	// fails because not oracle
	err = dposContract.SetMaxYearlyReward(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &SetMaxYearlyRewardRequest{
		MaxYearlyReward: &types.BigUInt{Value: *loom.NewBigUIntFromInt(100)},
	})
	require.Equal(t, errOnlyOracle, err)

	err = dposContract.SetMaxYearlyReward(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &SetMaxYearlyRewardRequest{
		MaxYearlyReward: &types.BigUInt{Value: *loom.NewBigUIntFromInt(100)},
	})
	require.NoError(t, err)

	stateResponse, err = dposContract.GetState(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &GetStateRequest{})
	assert.Equal(t, stateResponse.State.Params.MaxYearlyReward.Value.Int64(), int64(100))

	// set election cycle length

	// fails because not oracle
	err = dposContract.SetElectionCycle(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &SetElectionCycleRequest{
		ElectionCycle: int64(100),
	})
	require.Equal(t, errOnlyOracle, err)

	err = dposContract.SetElectionCycle(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &SetElectionCycleRequest{
		ElectionCycle: int64(100),
	})
	require.NoError(t, err)

	stateResponse, err = dposContract.GetState(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &GetStateRequest{})
	assert.Equal(t, stateResponse.State.Params.ElectionCycleLength, int64(100))
}

func TestRegisterWhitelistedCandidate(t *testing.T) {
	oraclePubKey, _ := hex.DecodeString(validatorPubKeyHex2)
	oracleAddr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(oraclePubKey),
	}

	pubKey, _ := hex.DecodeString(validatorPubKeyHex1)
	addr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey),
	}
	pubKey2, _ := hex.DecodeString(validatorPubKeyHex2)
	addr2 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey2),
	}
	pctx := plugin.CreateFakeContext(addr, addr)

	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(addr2, 2000000000000000000),
		},
	})

	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)
	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &InitRequest{
		Params: &Params{
			ValidatorCount: 21,
			OracleAddress:  oracleAddr.MarshalPB(),
		},
	})
	require.NoError(t, err)

	whitelistAmount := loom.BigUInt{big.NewInt(1000000000000)}
	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         10,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 1,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr)), &RegisterCandidateRequest{
		PubKey: pubKey,
	})
	require.Nil(t, err)

	err = dposContract.UnregisterCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(addr)), &UnregisterCandidateRequest{})
	require.Nil(t, err)

	registrationFee := &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}
	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr2)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr)), &RegisterCandidateRequest{
		PubKey: pubKey,
	})
	require.Nil(t, err)

	err = dposContract.RemoveWhitelistedCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &RemoveWhitelistedCandidateRequest{
		CandidateAddress: addr.MarshalPB(),
	})
	require.Nil(t, err)

	listResponse, err := dposContract.ListCandidates(contractpb.WrapPluginContext(dposCtx.WithSender(addr)), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, 2, len(listResponse.Candidates))

	err = dposContract.UnregisterCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(addr)), &UnregisterCandidateRequest{})
	require.Nil(t, err)

	listResponse, err = dposContract.ListCandidates(contractpb.WrapPluginContext(dposCtx.WithSender(addr)), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, 1, len(listResponse.Candidates))
}

func TestChangeFee(t *testing.T) {
	oldFee := uint64(100)
	newFee := uint64(1000)
	oraclePubKey, _ := hex.DecodeString(validatorPubKeyHex2)
	oracleAddr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(oraclePubKey),
	}

	dposContract := &DPOS{}

	pubKey, _ := hex.DecodeString(validatorPubKeyHex1)
	addr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey),
	}
	pctx := plugin.CreateFakeContext(addr, addr)

	// Deploy the coin contract (DPOS Init() will attempt to resolve it)
	coinContract := &coin.Coin{}
	_ = pctx.CreateContract(contractpb.MakePluginContract(coinContract))

	err := dposContract.Init(contractpb.WrapPluginContext(pctx.WithSender(oracleAddr)), &InitRequest{
		Params: &Params{
			ValidatorCount: 21,
			OracleAddress:  oracleAddr.MarshalPB(),
		},
	})
	require.Nil(t, err)

	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(pctx.WithSender(oracleAddr)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr.MarshalPB(),
					Amount:           &types.BigUInt{Value: loom.BigUInt{big.NewInt(1000000000000)}},
					LockTime:         10,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 1,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(pctx.WithSender(addr)), &RegisterCandidateRequest{
		PubKey: pubKey,
		Fee:    oldFee,
	})
	require.Nil(t, err)

	listResponse, err := dposContract.ListCandidates(contractpb.WrapPluginContext(pctx.WithSender(addr)), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, oldFee, listResponse.Candidates[0].Fee)
	assert.Equal(t, oldFee, listResponse.Candidates[0].NewFee)

	err = Elect(contractpb.WrapPluginContext(pctx.WithSender(addr)))
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(pctx.WithSender(addr)))
	require.Nil(t, err)

	// Fee should not reset
	listResponse, err = dposContract.ListCandidates(contractpb.WrapPluginContext(pctx.WithSender(addr)), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, oldFee, listResponse.Candidates[0].Fee)
	assert.Equal(t, oldFee, listResponse.Candidates[0].NewFee)

	err = dposContract.ChangeFee(contractpb.WrapPluginContext(pctx.WithSender(addr)), &d2types.ChangeCandidateFeeRequest{
		Fee: newFee,
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(pctx.WithSender(addr)))
	require.Nil(t, err)

	listResponse, err = dposContract.ListCandidates(contractpb.WrapPluginContext(pctx.WithSender(addr)), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, oldFee, listResponse.Candidates[0].Fee)
	assert.Equal(t, newFee, listResponse.Candidates[0].NewFee)

	err = Elect(contractpb.WrapPluginContext(pctx.WithSender(addr)))
	require.Nil(t, err)

	listResponse, err = dposContract.ListCandidates(contractpb.WrapPluginContext(pctx.WithSender(addr)), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, newFee, listResponse.Candidates[0].Fee)
	assert.Equal(t, newFee, listResponse.Candidates[0].NewFee)
}

func TestLockTimes(t *testing.T) {
	pubKey1, _ := hex.DecodeString(validatorPubKeyHex1)
	addr1 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey1),
	}
	oraclePubKey, _ := hex.DecodeString(validatorPubKeyHex2)
	oracleAddr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(oraclePubKey),
	}

	pctx := plugin.CreateFakeContext(addr1, addr1)

	// Deploy the coin contract (DPOS Init() will attempt to resolve it)
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 1000000000000000000),
			makeAccount(delegatorAddress2, 2000000000000000000),
			makeAccount(delegatorAddress3, 1000000000000000000),
			makeAccount(addr1, 9000000000000000000),
		},
	})

	registrationFee := &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}
	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)

	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &InitRequest{
		Params: &Params{
			ValidatorCount:          21,
			OracleAddress:           oracleAddr.MarshalPB(),
			RegistrationRequirement: registrationFee,
			ElectionCycleLength:     0, // Set to 1209600 in prod
		},
	})
	require.NoError(t, err)

	// Self-delegation check after registering via approve & registerCandidate
	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	now := uint64(dposCtx.Now().Unix())
	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey:       pubKey1,
		LocktimeTier: 1,
	})
	require.Nil(t, err)

	checkSelfDelegation, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: addr1.MarshalPB(),
	})
	selfDelegationLockTime := checkSelfDelegation.Delegation.LockTime

	assert.Equal(t, now+TierLocktimeMap[1], selfDelegationLockTime)
	assert.Equal(t, true, checkSelfDelegation.Delegation.LocktimeTier == 1)

	// make a delegation to candidate registered above

	approvalAmount := &types.BigUInt{Value: loom.BigUInt{big.NewInt(300)}}
	delegationAmount := &types.BigUInt{Value: loom.BigUInt{big.NewInt(100)}}
	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  approvalAmount,
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
		LocktimeTier:     2,
	})
	require.Nil(t, err)

	delegation1Response, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)

	d1LockTimeTier := delegation1Response.Delegation.LocktimeTier
	d1LockTime := delegation1Response.Delegation.LockTime
	assert.Equal(t, true, d1LockTimeTier == 2)
	assert.Equal(t, delegation1Response.Delegation.UpdateAmount.Value.Cmp(&delegationAmount.Value), 0)
	assert.Equal(t, delegation1Response.Delegation.Amount.Value.Cmp(loom.NewBigUIntFromInt(0)), 0)

	// Elections must happen so that we delegate again
	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Try delegating with a LockTime set to be less. It won't matter, since the existing locktime will be used.
	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
		LocktimeTier:     1,
	})
	require.Nil(t, err)

	delegation2Response, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	d2LockTime := delegation2Response.Delegation.LockTime
	d2LockTimeTier := delegation2Response.Delegation.LocktimeTier
	// New locktime should be the `now` value extended by the previous locktime
	assert.Equal(t, d2LockTime, now+d1LockTime)
	assert.Equal(t, true, d2LockTimeTier == 2)
	assert.Equal(t, delegation2Response.Delegation.UpdateAmount.Value.Cmp(&delegationAmount.Value), 0)
	expectedDelegation := common.BigZero()
	expectedDelegation.Mul(&delegationAmount.Value, loom.NewBigUIntFromInt(1))
	assert.Equal(t, delegation2Response.Delegation.Amount.Value.Cmp(expectedDelegation), 0)

	// Elections must happen so that we delegate again
	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Try delegating with a LockTime set to be bigger. It will overwrite the old locktime.
	now = uint64(dposCtx.Now().Unix())
	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
		LocktimeTier:     3,
	})
	require.Nil(t, err)

	delegation3Response, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	d3LockTime := delegation3Response.Delegation.LockTime
	d3LockTimeTier := delegation3Response.Delegation.LocktimeTier
	assert.Equal(t, delegation3Response.Delegation.UpdateAmount.Value.Cmp(&delegationAmount.Value), 0)
	expectedDelegation.Mul(&delegationAmount.Value, loom.NewBigUIntFromInt(2))
	assert.Equal(t, delegation3Response.Delegation.Amount.Value.Cmp(expectedDelegation), 0)

	// New locktime should be the `now` value extended by the new locktime
	assert.Equal(t, d3LockTime, now+d3LockTime)
	assert.Equal(t, true, d3LockTimeTier == 3)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Checking that delegator1 can't unbond before lock period is over
	err = dposContract.Unbond(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &UnbondRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.NotNil(t, err)

	// advancing contract time beyond the delegator1-addr1 lock period
	dposCtx.SetTime(dposCtx.Now().Add(time.Duration(now+d3LockTime+1) * time.Second))

	// Checking that delegator1 can unbond after lock period elapses
	err = dposContract.Unbond(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &UnbondRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.Nil(t, err)

	// Checking that delegator1 can't unbond twice in a single election period
	err = dposContract.Unbond(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &UnbondRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.NotNil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	delegationResponse, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)

	expectedDelegation.Mul(&delegationAmount.Value, loom.NewBigUIntFromInt(2))
	assert.True(t, delegationResponse.Delegation.Amount.Value.Cmp(expectedDelegation) == 0)
}

func TestDelegate(t *testing.T) {
	pubKey1, _ := hex.DecodeString(validatorPubKeyHex1)
	addr1 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey1),
	}
	oraclePubKey, _ := hex.DecodeString(validatorPubKeyHex2)
	oracleAddr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(oraclePubKey),
	}

	pctx := plugin.CreateFakeContext(addr1, addr1)

	// Deploy the coin contract (DPOS Init() will attempt to resolve it)
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 1000000000000000000),
			makeAccount(delegatorAddress2, 2000000000000000000),
			makeAccount(delegatorAddress3, 1000000000000000000),
			makeAccount(addr1, 1000000000000000000),
		},
	})

	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)
	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &InitRequest{
		Params: &Params{
			ValidatorCount: 21,
			OracleAddress:  oracleAddr.MarshalPB(),
		},
	})
	require.NoError(t, err)

	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr1.MarshalPB(),
					Amount:           &types.BigUInt{Value: loom.BigUInt{big.NewInt(1000000000000)}},
					LockTime:         10,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 1,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Error(t, err)

	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr1.MarshalPB(),
					Amount:           &types.BigUInt{Value: loom.BigUInt{big.NewInt(1000000000000)}},
					LockTime:         10,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 1,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	delegationAmount := &types.BigUInt{Value: loom.BigUInt{big.NewInt(100)}}
	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  delegationAmount,
	})
	require.Nil(t, err)

	response, err := coinContract.Allowance(contractpb.WrapPluginContext(coinCtx.WithSender(oracleAddr)), &coin.AllowanceRequest{
		Owner:   addr1.MarshalPB(),
		Spender: dposAddr.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegationAmount.Value.Int64(), response.Amount.Value.Int64())

	listResponse, err := dposContract.ListCandidates(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listResponse.Candidates), 1)
	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  delegationAmount,
	})
	require.Nil(t, err)

	// total rewards distribution should equal 0 before elections run
	rewardsResponse, err := dposContract.CheckRewards(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckRewardsRequest{})
	require.Nil(t, err)
	assert.True(t, rewardsResponse.TotalRewardDistribution.Value.Cmp(common.BigZero()) == 0)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// total rewards distribution should equal still be zero after first election
	rewardsResponse, err = dposContract.CheckRewards(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckRewardsRequest{})
	require.Nil(t, err)
	assert.True(t, rewardsResponse.TotalRewardDistribution.Value.Cmp(common.BigZero()) == 0)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.Nil(t, err)

	delegationResponse, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: addr1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.True(t, delegationResponse.Delegation.Amount.Value.Cmp(&delegationAmount.Value) == 0)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  delegationAmount,
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.Nil(t, err)

	// checking a non-existent delegation should result in an empty (amount = 0)
	// delegaiton being returned
	delegationResponse, err = dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: addr2.MarshalPB(),
	})
	require.Nil(t, err)
	assert.True(t, delegationResponse.Delegation.Amount.Value.Cmp(common.BigZero()) == 0)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  delegationAmount,
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// total rewards distribution should be greater than zero
	rewardsResponse, err = dposContract.CheckRewards(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckRewardsRequest{})
	require.Nil(t, err)
	assert.True(t, rewardsResponse.TotalRewardDistribution.Value.Cmp(common.BigZero()) > 0)

	err = dposContract.Unbond(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &UnbondRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	err = dposContract.Unbond(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &UnbondRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	err = dposContract.Unbond(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &UnbondRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: loom.BigUInt{big.NewInt(1)}},
	})
	assert.True(t, err != nil)

	// testing delegations to limbo validator
	err = dposContract.Redelegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &RedelegateRequest{
		FormerValidatorAddress: addr1.MarshalPB(),
		ValidatorAddress:       limboValidatorAddress.MarshalPB(),
		Amount:                 delegationAmount,
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	delegationResponse, err = dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.True(t, delegationResponse.Delegation.Amount.Value.Cmp(common.BigZero()) == 0)

	delegationResponse, err = dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDelegationRequest{
		ValidatorAddress: limboValidatorAddress.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.True(t, delegationResponse.Delegation.Amount.Value.Cmp(&delegationAmount.Value) == 0)
}

func TestRedelegate(t *testing.T) {
	pubKey1, _ := hex.DecodeString(validatorPubKeyHex1)
	addr1 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey1),
	}
	pubKey2, _ := hex.DecodeString(validatorPubKeyHex2)
	addr2 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey2),
	}
	pubKey3, _ := hex.DecodeString(validatorPubKeyHex3)
	addr3 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey3),
	}

	pctx := plugin.CreateFakeContext(addr1, addr1)

	// Deploy the coin contract (DPOS Init() will attempt to resolve it)
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 1000000000000000000),
			makeAccount(delegatorAddress2, 2000000000000000000),
			makeAccount(delegatorAddress3, 1000000000000000000),
			makeAccount(addr1, 1000000000000000000),
			makeAccount(addr2, 1000000000000000000),
			makeAccount(addr3, 1000000000000000000),
		},
	})

	registrationFee := loom.BigZeroPB()

	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)
	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &InitRequest{
		Params: &Params{
			ValidatorCount:          21,
			RegistrationRequirement: registrationFee,
		},
	})
	require.NoError(t, err)

	// Registering 3 candidates
	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr3)), &RegisterCandidateRequest{
		PubKey: pubKey3,
	})
	require.Nil(t, err)

	listResponse, err := dposContract.ListCandidates(contractpb.WrapPluginContext(dposCtx), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listResponse.Candidates), 3)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Verifying that with registration fee = 0, none of the 3 registered candidates are elected validators
	listValidatorsResponse, err := dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 0)

	delegationAmount := loom.NewBigUIntFromInt(10000000)
	smallDelegationAmount := loom.NewBigUIntFromInt(1000000)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Verifying that addr1 was elected sole validator
	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 1)
	assert.True(t, listValidatorsResponse.Statistics[0].Address.Local.Compare(addr1.Local) == 0)

	// checking that redelegation fails with 0 amount
	err = dposContract.Redelegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &RedelegateRequest{
		FormerValidatorAddress: addr1.MarshalPB(),
		ValidatorAddress:       addr2.MarshalPB(),
		Amount:                 loom.BigZeroPB(),
	})
	require.NotNil(t, err)

	// redelegating sole delegation to validator addr2
	err = dposContract.Redelegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &RedelegateRequest{
		FormerValidatorAddress: addr1.MarshalPB(),
		ValidatorAddress:       addr2.MarshalPB(),
		Amount:                 &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	// Redelegation takes effect within a single election period
	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Verifying that addr2 was elected sole validator
	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 1)
	assert.True(t, listValidatorsResponse.Statistics[0].Address.Local.Compare(addr2.Local) == 0)

	// redelegating sole delegation to validator addr3
	err = dposContract.Redelegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &RedelegateRequest{
		FormerValidatorAddress: addr2.MarshalPB(),
		ValidatorAddress:       addr3.MarshalPB(),
		Amount:                 &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	// Redelegation takes effect within a single election period
	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Verifying that addr3 was elected sole validator
	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 1)
	assert.True(t, listValidatorsResponse.Statistics[0].Address.Local.Compare(addr3.Local) == 0)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress2)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	// adding 2nd delegation from 2nd delegator in order to elect a second validator
	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// checking that the 2nd validator (addr1) was elected in addition to add3
	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 2)

	// delegator 1 removes delegation to limbo
	err = dposContract.Redelegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &RedelegateRequest{
		FormerValidatorAddress: addr3.MarshalPB(),
		ValidatorAddress:       limboValidatorAddress.MarshalPB(),
		Amount:                 &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Verifying that addr1 was elected sole validator AFTER delegator1 redelegated to limbo validator
	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 1)
	assert.True(t, listValidatorsResponse.Statistics[0].Address.Local.Compare(addr1.Local) == 0)

	// Checking that redelegaiton of a negative amount is rejected
	err = dposContract.Redelegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &RedelegateRequest{
		FormerValidatorAddress: addr1.MarshalPB(),
		ValidatorAddress:       addr2.MarshalPB(),
		Amount:                 &types.BigUInt{Value: *loom.NewBigUIntFromInt(-1000)},
	})
	require.NotNil(t, err)

	// Checking that redelegaiton of an amount greater than the total delegation is rejected
	err = dposContract.Redelegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &RedelegateRequest{
		FormerValidatorAddress: addr1.MarshalPB(),
		ValidatorAddress:       addr2.MarshalPB(),
		Amount:                 &types.BigUInt{Value: *loom.NewBigUIntFromInt(100000000)},
	})
	require.NotNil(t, err)

	// splitting delegator2's delegation to 2nd validator
	err = dposContract.Redelegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &RedelegateRequest{
		FormerValidatorAddress: addr1.MarshalPB(),
		ValidatorAddress:       addr2.MarshalPB(),
		Amount:                 &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	// splitting delegator2's delegation to 3rd validator
	err = dposContract.Redelegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &RedelegateRequest{
		FormerValidatorAddress: addr1.MarshalPB(),
		ValidatorAddress:       addr3.MarshalPB(),
		Amount:                 &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// checking that all 3 candidates have been elected validators
	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 3)
}

func TestReward(t *testing.T) {
	// set elect time in params to one second for easy calculations
	delegationAmount := loom.BigUInt{big.NewInt(10000000000000)}
	cycleLengthSeconds := int64(100)
	params := Params{
		ElectionCycleLength: cycleLengthSeconds,
		MaxYearlyReward:     &types.BigUInt{Value: *scientificNotation(defaultMaxYearlyReward, tokenDecimals)},
	}
	statistic := ValidatorStatistic{
		DistributionTotal: &types.BigUInt{Value: loom.BigUInt{big.NewInt(0)}},
		DelegationTotal:   &types.BigUInt{Value: delegationAmount},
	}
	for i := int64(0); i < yearSeconds; i = i + cycleLengthSeconds {
		rewardValidator(&statistic, &params, *common.BigZero(), false)
	}

	// checking that distribution is roughtly equal to 5% of delegation after one year
	assert.Equal(t, statistic.DistributionTotal.Value.Cmp(&loom.BigUInt{big.NewInt(490000000000)}), 1)
	assert.Equal(t, statistic.DistributionTotal.Value.Cmp(&loom.BigUInt{big.NewInt(510000000000)}), -1)
}

func TestElectWhitelists(t *testing.T) {
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

	pubKey4, _ := hex.DecodeString(validatorPubKeyHex4)
	addr4 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey4),
	}

	pubKey5, _ := hex.DecodeString(validatorPubKeyHex5)
	addr5 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey5),
	}

	// Init the coin balances
	var startTime int64 = 100000
	pctx := plugin.CreateFakeContext(delegatorAddress1, loom.Address{}).WithBlock(loom.BlockHeader{
		ChainID: chainID,
		Time:    startTime,
	})
	coinAddr := pctx.CreateContract(coin.Contract)

	coinContract := &coin.Coin{}
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 1e18),
			makeAccount(delegatorAddress2, 20),
			makeAccount(delegatorAddress3, 10),
		},
	})

	// create dpos contract
	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)

	// transfer coins to reward fund
	amount := big.NewInt(10000000)
	amount.Mul(amount, big.NewInt(1e18))
	err := coinContract.Transfer(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.TransferRequest{
		To:     dposAddr.MarshalPB(),
		Amount: &types.BigUInt{Value: loom.BigUInt{amount}},
	})
	require.Nil(t, err)

	// Enable the feature flag and check that the whitelist rules get applied corectly
	dposCtx.SetFeature(loomchain.DPOSVersion2_1, true)
	require.True(t, dposCtx.FeatureEnabled(loomchain.DPOSVersion2_1, false))

	cycleLengthSeconds := int64(100)
	// Init the dpos contract
	err = dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &InitRequest{
		Params: &Params{
			CoinContractAddress: coinAddr.MarshalPB(),
			ValidatorCount:      5,
			ElectionCycleLength: cycleLengthSeconds,
			OracleAddress:       addr1.MarshalPB(),
			MaxYearlyReward:     &types.BigUInt{Value: *scientificNotation(defaultMaxYearlyReward, tokenDecimals)},
		},
	})
	require.Nil(t, err)

	whitelistAmount := loom.BigUInt{big.NewInt(1000000000000)}

	// Whitelist with locktime tier 0, which should use 5% of rewards
	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr1.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         0,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 1,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	// Whitelist with locktime tier 1, which should use 7.5% of rewards
	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr2.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         1,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 2,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	// Whitelist with locktime tier 2, which should use 10% of rewards
	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr3.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         2,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 3,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	// Whitelist with locktime tier 3, which should use 20% of rewards
	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr4.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         3,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 4,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	// Whitelist with a random locktime, which should use the 5% rewards
	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr5.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         12321451,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 5,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})

	// Register the 5 validators

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr3)), &RegisterCandidateRequest{
		PubKey: pubKey3,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr4)), &RegisterCandidateRequest{
		PubKey: pubKey4,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr5)), &RegisterCandidateRequest{
		PubKey: pubKey5,
	})
	require.Nil(t, err)

	// Check that they were registered properly
	listCandidatesResponse, err := dposContract.ListCandidates(contractpb.WrapPluginContext(dposCtx), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listCandidatesResponse.Candidates), 5)

	listValidatorsResponse, err := dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 0)

	// Elect them
	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 5)

	// Do a bunch of elections that correspond to 1/100th of a year
	for i := int64(0); i < yearSeconds/100; i = i + cycleLengthSeconds {
		err = Elect(contractpb.WrapPluginContext(dposCtx))
		require.Nil(t, err)
		dposCtx.SetTime(dposCtx.Now().Add(time.Duration(cycleLengthSeconds) * time.Second))
	}

	checkResponse, err := dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDistributionRequest{Address: addr1.MarshalPB()})
	require.Nil(t, err)
	// checking that rewards are roughtly equal to 0.5% of delegation after one year
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(490000000)}), 1)
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(510000000)}), -1)

	checkResponse, err = dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDistributionRequest{Address: addr2.MarshalPB()})
	require.Nil(t, err)
	// checking that rewards are roughtly equal to 0.75% of delegation after one year
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(740000000)}), 1)
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(760000000)}), -1)

	checkResponse, err = dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDistributionRequest{Address: addr3.MarshalPB()})
	require.Nil(t, err)
	// checking that rewards are roughtly equal to 1% of delegation after one year
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(990000000)}), 1)
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(1000000000)}), -1)

	checkResponse, err = dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDistributionRequest{Address: addr4.MarshalPB()})
	require.Nil(t, err)
	// checking that rewards are roughtly equal to 2% of delegation after one year
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(1990000000)}), 1)
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(2000000000)}), -1)

	checkResponse, err = dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDistributionRequest{Address: addr5.MarshalPB()})
	require.Nil(t, err)
	// checking that rewards are roughtly equal to 0.5% of delegation after one year
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(490000000)}), 1)
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(510000000)}), -1)

	// Let's withdraw rewards and see how the balances change.
	_, err = dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &ClaimDistributionRequest{
		WithdrawalAddress: addr1.MarshalPB(),
	})
	require.Nil(t, err)

	_, err = dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &ClaimDistributionRequest{
		WithdrawalAddress: addr2.MarshalPB(),
	})
	require.Nil(t, err)

	_, err = dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr3)), &ClaimDistributionRequest{
		WithdrawalAddress: addr3.MarshalPB(),
	})
	require.Nil(t, err)

	_, err = dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr4)), &ClaimDistributionRequest{
		WithdrawalAddress: addr4.MarshalPB(),
	})
	require.Nil(t, err)

	_, err = dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr5)), &ClaimDistributionRequest{
		WithdrawalAddress: addr5.MarshalPB(),
	})
	require.Nil(t, err)

	balanceAfterClaim, err := coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx), &coin.BalanceOfRequest{
		Owner: addr1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, balanceAfterClaim.Balance.Value.Cmp(&loom.BigUInt{big.NewInt(490000000)}), 1)
	assert.Equal(t, balanceAfterClaim.Balance.Value.Cmp(&loom.BigUInt{big.NewInt(510000000)}), -1)

	balanceAfterClaim, err = coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx), &coin.BalanceOfRequest{
		Owner: addr2.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, balanceAfterClaim.Balance.Value.Cmp(&loom.BigUInt{big.NewInt(740000000)}), 1)
	assert.Equal(t, balanceAfterClaim.Balance.Value.Cmp(&loom.BigUInt{big.NewInt(760000000)}), -1)

	balanceAfterClaim, err = coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx), &coin.BalanceOfRequest{
		Owner: addr3.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, balanceAfterClaim.Balance.Value.Cmp(&loom.BigUInt{big.NewInt(990000000)}), 1)
	assert.Equal(t, balanceAfterClaim.Balance.Value.Cmp(&loom.BigUInt{big.NewInt(1000000000)}), -1)

	balanceAfterClaim, err = coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx), &coin.BalanceOfRequest{
		Owner: addr4.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, balanceAfterClaim.Balance.Value.Cmp(&loom.BigUInt{big.NewInt(1990000000)}), 1)
	assert.Equal(t, balanceAfterClaim.Balance.Value.Cmp(&loom.BigUInt{big.NewInt(2000000000)}), -1)

	balanceAfterClaim, err = coinContract.BalanceOf(contractpb.WrapPluginContext(coinCtx), &coin.BalanceOfRequest{
		Owner: addr5.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, balanceAfterClaim.Balance.Value.Cmp(&loom.BigUInt{big.NewInt(490000000)}), 1)
	assert.Equal(t, balanceAfterClaim.Balance.Value.Cmp(&loom.BigUInt{big.NewInt(510000000)}), -1)

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
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 130),
			makeAccount(delegatorAddress2, 20),
			makeAccount(delegatorAddress3, 10),
		},
	})

	// create dpos contract
	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)

	// transfer coins to reward fund
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(19), nil)
	coinContract.Transfer(contractpb.WrapPluginContext(coinCtx), &coin.TransferRequest{
		To: dposAddr.MarshalPB(),
		Amount: &types.BigUInt{
			Value: common.BigUInt{amount},
		},
	})

	// Init the dpos contract
	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &InitRequest{
		Params: &Params{
			CoinContractAddress: coinAddr.MarshalPB(),
			ValidatorCount:      2,
			ElectionCycleLength: 0,
			OracleAddress:       addr1.MarshalPB(),
		},
	})
	require.Nil(t, err)

	whitelistAmount := loom.BigUInt{big.NewInt(1000000000000)}

	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr1.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         10,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 1,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr2.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         10,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 2,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr3.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         10,
				}},
				Meta: &d2types.BatchRequestMetaV2{
					BlockNumber: 3,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr3)), &RegisterCandidateRequest{
		PubKey: pubKey3,
	})
	require.Nil(t, err)

	listCandidatesResponse, err := dposContract.ListCandidates(contractpb.WrapPluginContext(dposCtx), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listCandidatesResponse.Candidates), 3)

	listValidatorsResponse, err := dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 0)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 2)

	for i := 0; i < 10; i = i + 1 {
		err = Elect(contractpb.WrapPluginContext(dposCtx))
		require.Nil(t, err)
		claimResponse, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &ClaimDistributionRequest{
			WithdrawalAddress: addr1.MarshalPB(),
		})
		require.Nil(t, err)
		assert.Equal(t, claimResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)
	}

	claimResponse, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &ClaimDistributionRequest{
		WithdrawalAddress: addr1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, claimResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	// Change WhitelistAmount and verify that it got changed correctly
	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	validator := listValidatorsResponse.Statistics[0]
	assert.Equal(t, whitelistAmount, validator.WhitelistAmount.Value)

	newWhitelistAmount := loom.BigUInt{big.NewInt(2000000000000)}

	// only oracle
	err = dposContract.ChangeWhitelistAmount(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &ChangeWhitelistAmountRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: newWhitelistAmount},
	})
	require.Equal(t, errOnlyOracle, err)

	err = dposContract.ChangeWhitelistAmount(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &ChangeWhitelistAmountRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: newWhitelistAmount},
	})
	require.Nil(t, err)

	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	validator = listValidatorsResponse.Statistics[0]
	assert.Equal(t, newWhitelistAmount, validator.WhitelistAmount.Value)
}

func TestValidatorRewards(t *testing.T) {
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
	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)

	// transfer coins to reward fund
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(19), nil)
	coinContract.Transfer(contractpb.WrapPluginContext(coinCtx), &coin.TransferRequest{
		To: dposAddr.MarshalPB(),
		Amount: &types.BigUInt{
			Value: common.BigUInt{amount},
		},
	})

	// Init the dpos contract
	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &InitRequest{
		Params: &Params{
			CoinContractAddress: coinAddr.MarshalPB(),
			ValidatorCount:      10,
			ElectionCycleLength: 0,
		},
	})
	require.Nil(t, err)

	registrationFee := &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr2)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr3)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr3)), &RegisterCandidateRequest{
		PubKey: pubKey3,
	})
	require.Nil(t, err)

	listCandidatesResponse, err := dposContract.ListCandidates(contractpb.WrapPluginContext(dposCtx), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listCandidatesResponse.Candidates), 3)

	listValidatorsResponse, err := dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 0)
	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 3)

	// Two delegators delegate 1/2 and 1/4 of a registration fee respectively
	smallDelegationAmount := loom.NewBigUIntFromInt(0)
	smallDelegationAmount.Div(&registrationFee.Value, loom.NewBigUIntFromInt(4))
	largeDelegationAmount := loom.NewBigUIntFromInt(0)
	largeDelegationAmount.Div(&registrationFee.Value, loom.NewBigUIntFromInt(2))

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress2)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *largeDelegationAmount},
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *largeDelegationAmount},
	})
	require.Nil(t, err)

	for i := 0; i < 10000; i = i + 1 {
		err = Elect(contractpb.WrapPluginContext(dposCtx))
		require.Nil(t, err)
	}

	checkResponse, err := dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDistributionRequest{Address: addr1.MarshalPB()})
	require.Nil(t, err)
	assert.Equal(t, checkResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	claimResponse, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &ClaimDistributionRequest{
		WithdrawalAddress: addr1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, claimResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)
	assert.Equal(t, claimResponse.Amount.Value.Cmp(&checkResponse.Amount.Value), 0)

	delegator1Claim, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegator1Claim.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	delegator2Claim, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress2.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegator2Claim.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	halvedDelegator2Claim := loom.NewBigUIntFromInt(0)
	halvedDelegator2Claim.Div(&delegator2Claim.Amount.Value, loom.NewBigUIntFromInt(2))
	difference := loom.NewBigUIntFromInt(0)
	difference.Sub(&delegator1Claim.Amount.Value, halvedDelegator2Claim)

	// Checking that Delegator2's claim is almost exactly half of Delegator1's claim
	maximumDifference := scientificNotation(1, tokenDecimals)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)
}

func TestRewardTiers(t *testing.T) {
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

	// create dpos contract
	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)

	// transfer coins to reward fund
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(19), nil)
	coinContract.Transfer(contractpb.WrapPluginContext(coinCtx), &coin.TransferRequest{
		To: dposAddr.MarshalPB(),
		Amount: &types.BigUInt{
			Value: common.BigUInt{amount},
		},
	})

	// Init the dpos contract
	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &InitRequest{
		Params: &Params{
			CoinContractAddress: coinAddr.MarshalPB(),
			ValidatorCount:      10,
			ElectionCycleLength: 0,
		},
	})
	require.Nil(t, err)

	registrationFee := &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr2)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr3)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr3)), &RegisterCandidateRequest{
		PubKey: pubKey3,
	})
	require.Nil(t, err)

	listCandidatesResponse, err := dposContract.ListCandidates(contractpb.WrapPluginContext(dposCtx), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listCandidatesResponse.Candidates), 3)

	listValidatorsResponse, err := dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 0)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 3)

	tinyDelegationAmount := scientificNotation(1, tokenDecimals) // 1 token
	smallDelegationAmount := loom.NewBigUIntFromInt(0)
	smallDelegationAmount.Div(&registrationFee.Value, loom.NewBigUIntFromInt(4))
	largeDelegationAmount := loom.NewBigUIntFromInt(0)
	largeDelegationAmount.Div(&registrationFee.Value, loom.NewBigUIntFromInt(2))

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	// LocktimeTier should default to 0 for delegatorAddress1
	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress2)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *smallDelegationAmount},
		LocktimeTier:     2,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress3)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress3)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *smallDelegationAmount},
		LocktimeTier:     3,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress4)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress4)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *smallDelegationAmount},
		LocktimeTier:     1,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress5)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *largeDelegationAmount},
	})
	require.Nil(t, err)

	// Though Delegator5 delegates to addr2 and not addr1 like the rest of the
	// delegators, he should still receive the same rewards proportional to his
	// delegation parameters
	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress5)), &DelegateRequest{
		ValidatorAddress: addr2.MarshalPB(),
		Amount:           &types.BigUInt{Value: *largeDelegationAmount},
		LocktimeTier:     2,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress6)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *tinyDelegationAmount},
	})
	require.Nil(t, err)

	// by delegating a very small amount, delegator6 demonstrates that
	// delegators can contribute far less than 0.01% of a validator's total
	// delegation and still be rewarded - ONLY AFTER THE FEATURE FLAG FOR v2.1 is enabled!
	err = dposContract.Delegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress6)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *tinyDelegationAmount},
	})
	require.Nil(t, err)

	for i := 0; i < 10000; i = i + 1 {
		err = Elect(contractpb.WrapPluginContext(dposCtx))
		require.Nil(t, err)
	}

	// Test that the delegator got 0 rewards, because of the <0.01% rewards bug.
	delegator6Claim, err := dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress6)),
		&CheckDistributionRequest{Address: delegatorAddress6.MarshalPB()},
	)
	require.Nil(t, err)
	assert.Equal(t, delegator6Claim.Amount.Value.Cmp(common.BigZero()), 0)

	claimResponse, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &ClaimDistributionRequest{
		WithdrawalAddress: addr1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, claimResponse.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	delegator1Claim, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegator1Claim.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	delegator2Claim, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress2.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegator2Claim.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	delegator3Claim, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress3)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress3.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegator3Claim.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	delegator4Claim, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress4)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress4.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegator4Claim.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	delegator5Claim, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress5)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress5.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegator5Claim.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	maximumDifference := scientificNotation(1, tokenDecimals)
	difference := loom.NewBigUIntFromInt(0)

	// Checking that Delegator2's claim is almost exactly twice Delegator1's claim
	scaledDelegator1Claim := CalculateFraction(*loom.NewBigUIntFromInt(20000), delegator1Claim.Amount.Value, true)
	difference.Sub(&scaledDelegator1Claim, &delegator2Claim.Amount.Value)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)

	// Checking that Delegator3's & Delegator5's claim is almost exactly four times Delegator1's claim
	scaledDelegator1Claim = CalculateFraction(*loom.NewBigUIntFromInt(40000), delegator1Claim.Amount.Value, true)

	difference.Sub(&scaledDelegator1Claim, &delegator3Claim.Amount.Value)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)

	difference.Sub(&scaledDelegator1Claim, &delegator5Claim.Amount.Value)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)

	// Checking that Delegator4's claim is almost exactly 1.5 times Delegator1's claim
	scaledDelegator1Claim = CalculateFraction(*loom.NewBigUIntFromInt(15000), delegator1Claim.Amount.Value, true)
	difference.Sub(&scaledDelegator1Claim, &delegator4Claim.Amount.Value)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)

	// Testing total delegation functionality

	totalDelegationResponse, err := dposContract.TotalDelegation(contractpb.WrapPluginContext(dposCtx), &TotalDelegationRequest{
		DelegatorAddress: delegatorAddress3.MarshalPB(),
	})
	require.Nil(t, err)
	assert.True(t, totalDelegationResponse.Amount.Value.Cmp(smallDelegationAmount) == 0)
	expectedWeightedAmount := CalculateFraction(*loom.NewBigUIntFromInt(40000), *smallDelegationAmount, true)
	assert.True(t, totalDelegationResponse.WeightedAmount.Value.Cmp(&expectedWeightedAmount) == 0)

	// Enable the feature flag and check that the delegator receives rewards!
	dposCtx.SetFeature(loomchain.DPOSVersion2_1, true)
	require.True(t, dposCtx.FeatureEnabled(loomchain.DPOSVersion2_1, false))

	for i := 0; i < 10000; i = i + 1 {
		err = Elect(contractpb.WrapPluginContext(dposCtx))
		require.Nil(t, err)
	}

	// Test that the delegator got >0 rewards -- v2.1 bug is fixed now since the flag got enabled
	delegator6ClaimFixed, err := dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress6)),
		&CheckDistributionRequest{Address: delegatorAddress6.MarshalPB()},
	)
	require.Nil(t, err)
	assert.Equal(t, delegator6ClaimFixed.Amount.Value.Cmp(common.BigZero()), 1)
}

// Besides reward cap functionality, this also demostrates 0-fee candidate registration
func TestRewardCap(t *testing.T) {
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
	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)

	// transfer coins to reward fund
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(19), nil)
	coinContract.Transfer(contractpb.WrapPluginContext(coinCtx), &coin.TransferRequest{
		To: dposAddr.MarshalPB(),
		Amount: &types.BigUInt{
			Value: common.BigUInt{amount},
		},
	})

	registrationFee := loom.BigZeroPB()

	// Init the dpos contract
	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &InitRequest{
		Params: &Params{
			CoinContractAddress: coinAddr.MarshalPB(),
			ValidatorCount:      10,
			ElectionCycleLength: 0,
			MaxYearlyReward:     &types.BigUInt{Value: *scientificNotation(100, tokenDecimals)},
			// setting registration fee to zero for easy calculations using delegations alone
			RegistrationRequirement: registrationFee,
		},
	})

	require.Nil(t, err)
	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr3)), &RegisterCandidateRequest{
		PubKey: pubKey3,
	})
	require.Nil(t, err)

	listCandidatesResponse, err := dposContract.ListCandidates(contractpb.WrapPluginContext(dposCtx), &ListCandidateRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listCandidatesResponse.Candidates), 3)

	listValidatorsResponse, err := dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 0)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 0)

	delegationAmount := scientificNotation(1000, tokenDecimals)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress2)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &DelegateRequest{
		ValidatorAddress: addr2.MarshalPB(),
		Amount:           &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	// With a default yearly reward of 5% of one's token holdings, the two
	// delegators should reach their rewards limits by both delegating exactly
	// 1000, or 2000 combined since 2000 = 100 (the max yearly reward) / 0.05

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 2)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	delegator1Claim, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegator1Claim.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	delegator2Claim, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress2.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegator2Claim.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	//                           |---- this 2 is the election cycle length used when,
	//    v--- delegationAmount  v     for testing, a 0-sec election time is set
	// ((1000 * 10**18) * 0.05 * 2) / (365 * 24 * 3600) = 3.1709791983764585e12
	expectedAmount := loom.NewBigUIntFromInt(3170979198376)
	assert.Equal(t, *expectedAmount, delegator2Claim.Amount.Value)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress3)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress3)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *delegationAmount},
	})
	require.Nil(t, err)

	// run one election to get Delegator3 elected as a validator
	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// run another election to get Delegator3 his first reward distribution
	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	delegator3Claim, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress3)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress3.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, delegator3Claim.Amount.Value.Cmp(&loom.BigUInt{big.NewInt(0)}), 1)

	// verifiying that claim is smaller than what was given when delegations
	// were smaller and below max yearly reward cap.
	// delegator3Claim should be ~2/3 of delegator2Claim
	assert.Equal(t, delegator2Claim.Amount.Value.Cmp(&delegator3Claim.Amount.Value), 1)
	scaledDelegator3Claim := CalculateFraction(*loom.NewBigUIntFromInt(15000), delegator3Claim.Amount.Value, true)
	difference := common.BigZero()
	difference.Sub(&scaledDelegator3Claim, &delegator2Claim.Amount.Value)
	// amounts must be within 3 * 10^-18 tokens of each other to be correct
	maximumDifference := loom.NewBigUIntFromInt(3)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)
}

func TestPostLocktimeRewards(t *testing.T) {
	pubKey1, _ := hex.DecodeString(validatorPubKeyHex1)
	addr1 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey1),
	}
	oraclePubKey, _ := hex.DecodeString(validatorPubKeyHex2)
	oracleAddr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(oraclePubKey),
	}

	pctx := plugin.CreateFakeContext(addr1, addr1)

	// Deploy the coin contract (DPOS Init() will attempt to resolve it)
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 1000000000000000000),
			makeAccount(delegatorAddress2, 2000000000000000000),
			makeAccount(delegatorAddress3, 1000000000000000000),
			makeAccount(addr1, 9000000000000000000),
		},
	})

	registrationFee := &types.BigUInt{Value: *scientificNotation(defaultRegistrationRequirement, tokenDecimals)}
	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)

	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(oracleAddr)), &InitRequest{
		Params: &Params{
			ValidatorCount:          21,
			OracleAddress:           oracleAddr.MarshalPB(),
			RegistrationRequirement: registrationFee,
			ElectionCycleLength:     0, // Set to 1209600 in prod
		},
	})
	require.NoError(t, err)

	// Self-delegation check after registering via approve & registerCandidate
	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	now := uint64(dposCtx.Now().Unix())
	err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey:       pubKey1,
		LocktimeTier: 1,
		Fee:          0,
	})
	require.Nil(t, err)

	checkSelfDelegation, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: addr1.MarshalPB(),
	})
	selfDelegationLockTime := checkSelfDelegation.Delegation.LockTime

	assert.Equal(t, now+TierLocktimeMap[1], selfDelegationLockTime)
	assert.Equal(t, true, checkSelfDelegation.Delegation.LocktimeTier == 1)
	assert.Equal(t, checkSelfDelegation.Delegation.Amount.Value.Cmp(common.BigZero()), 0)
	assert.Equal(t, checkSelfDelegation.Delegation.UpdateAmount.Value.Cmp(&registrationFee.Value), 0)

	// make a delegation to candidate registered above

	approvalAmount := &types.BigUInt{Value: *scientificNotation(300000, tokenDecimals)}
	delegationAmount := &types.BigUInt{Value: *scientificNotation(100000, tokenDecimals)}
	unbondAmount := &types.BigUInt{Value: *scientificNotation(10000, tokenDecimals)}
	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  approvalAmount,
	})
	require.Nil(t, err)

	err = dposContract.Delegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
		LocktimeTier:     2,
	})
	require.Nil(t, err)

	delegation1Response, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)

	d1LockTimeTier := delegation1Response.Delegation.LocktimeTier
	d1LockTime := delegation1Response.Delegation.LockTime
	assert.Equal(t, delegation1Response.Delegation.Amount.Value.Cmp(common.BigZero()), 0)
	assert.Equal(t, delegation1Response.Delegation.UpdateAmount.Value.Cmp(&delegationAmount.Value), 0)
	assert.Equal(t, true, d1LockTimeTier == 2)

	rewardsResponse, err := dposContract.CheckRewards(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckRewardsRequest{})
	require.Nil(t, err)
	assert.True(t, rewardsResponse.TotalRewardDistribution.Value.Cmp(common.BigZero()) == 0)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	rewardsResponse, err = dposContract.CheckRewards(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckRewardsRequest{})
	require.Nil(t, err)
	assert.True(t, rewardsResponse.TotalRewardDistribution.Value.Cmp(common.BigZero()) == 0)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	rewardsResponse, err = dposContract.CheckRewards(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckRewardsRequest{})
	assert.True(t, rewardsResponse.TotalRewardDistribution.Value.Cmp(common.BigZero()) != 0)
	require.Nil(t, err)

	checkDistributionResponse, err := dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDistributionRequest{Address: addr1.MarshalPB()})
	require.Nil(t, err)
	assert.True(t, checkDistributionResponse.Amount.Value.Cmp(common.BigZero()) == 1)

	checkDistributionResponse, err = dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckDistributionRequest{Address: delegatorAddress1.MarshalPB()})
	require.Nil(t, err)
	// recording single-election distribution amount prior to lockup expiry
	priorRewardValue := checkDistributionResponse.Amount.Value
	assert.True(t, priorRewardValue.Cmp(common.BigZero()) == 1)

	claimResponse, err := dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.True(t, claimResponse.Amount.Value.Cmp(common.BigZero()) == 1)
	assert.True(t, claimResponse.Amount.Value.Cmp(&priorRewardValue) == 0)

	// Try delegating with a LockTime set to be bigger. It will overwrite the old locktime.
	now = uint64(dposCtx.Now().Unix())

	// Checking that delegator1 can't unbond before lock period is over
	err = dposContract.Unbond(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &UnbondRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.NotNil(t, err)

	// advancing contract time beyond the delegator1-addr1 lock period
	dposCtx.SetTime(dposCtx.Now().Add(time.Duration(now+d1LockTime+1) * time.Second))

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Checking that for two election periods after lockup expires,
	// a delegator's rewards do not change
	checkDistributionResponse, err = dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckDistributionRequest{Address: delegatorAddress1.MarshalPB()})
	require.Nil(t, err)
	assert.True(t, checkDistributionResponse.Amount.Value.Cmp(common.BigZero()) == 1)

	claimResponse, err = dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.True(t, claimResponse.Amount.Value.Cmp(common.BigZero()) == 1)
	assert.True(t, claimResponse.Amount.Value.Cmp(&priorRewardValue) == 0)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	checkDistributionResponse, err = dposContract.CheckDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckDistributionRequest{Address: delegatorAddress1.MarshalPB()})
	require.Nil(t, err)
	assert.True(t, checkDistributionResponse.Amount.Value.Cmp(common.BigZero()) == 1)

	claimResponse, err = dposContract.ClaimDistribution(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &ClaimDistributionRequest{
		WithdrawalAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.True(t, claimResponse.Amount.Value.Cmp(common.BigZero()) == 1)
	assert.True(t, claimResponse.Amount.Value.Cmp(&priorRewardValue) == 0)

	// checking that listDelegations returns expected values
	listDelegationsResponse, err := dposContract.ListDelegations(contractpb.WrapPluginContext(dposCtx), &ListDelegationsRequest{Candidate: addr1.MarshalPB()})
	require.Nil(t, err)
	assert.Equal(t, len(listDelegationsResponse.Delegations), 2)
	expectedTotalDelegationAmount := common.BigZero()
	expectedTotalDelegationAmount = expectedTotalDelegationAmount.Add(&delegationAmount.Value, &registrationFee.Value)
	assert.True(t, listDelegationsResponse.DelegationTotal.Value.Cmp(expectedTotalDelegationAmount) == 0)

	// Checking that delegator1 can unbond after lock period elapses
	err = dposContract.Unbond(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &UnbondRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           unbondAmount,
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Check that lockup time does not change after a partial unbond
	checkDelegation, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.True(t, checkDelegation.Delegation.LockTime == d1LockTime)

	checkAllDelegations, err := dposContract.CheckAllDelegations(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckAllDelegationsRequest{
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	assert.Equal(t, 1, len(checkAllDelegations.Delegations))
	assert.True(t, checkAllDelegations.Delegations[0].LockTime == d1LockTime)
}

// due to a chain reorg, a particular delegation is 2x the amount it should be
func TestDedoublingDelegation(t *testing.T) {
	originalAmount := loom.NewBigUIntFromInt(100)
	doubledDelegation := Delegation{
		Validator: doubledValidator.MarshalPB(),
		Delegator: doubledDelegator.MarshalPB(),
		Amount:    &types.BigUInt{Value: *originalAmount},
	}
	adjustedAmount := adjustDoubledDelegationAmount(doubledDelegation)
	expectedAmount := common.BigZero()
	expectedAmount.Div(originalAmount, loom.NewBigUIntFromInt(2))
	assert.True(t, expectedAmount.Cmp(&adjustedAmount.Value) == 0)

	// test that adjustDoubledDelegationAmount does not halve the amount of a delegation which does not match the doubledDelegator & doubledValidator
	nonDoubledDelegation := Delegation{
		Validator: delegatorAddress1.MarshalPB(),
		Delegator: doubledDelegator.MarshalPB(),
		Amount:    &types.BigUInt{Value: *originalAmount},
	}
	adjustedAmount = adjustDoubledDelegationAmount(nonDoubledDelegation)
	assert.True(t, adjustedAmount.Value.Cmp(originalAmount) == 0)
}

// after we migrate we want to have all delegations that were in plasma-* nodes, on plasma-0
func TestPlasmaDelegationMigration(t *testing.T) {
	expectedValidator := PlasmaValidators[0]
	someDelegator := loom.MustParseAddress("default:0xDc93E46f6d22D47De9D7E6d26ce8c3b7A13d89Cb")
	someAmount := types.BigUInt{Value: *loom.NewBigUIntFromInt(100)}

	// check that all plasma validators get reset to plasma-0
	for _, v := range PlasmaValidators {
		migratedDelegation := Delegation{
			Validator: v.MarshalPB(),
			Delegator: someDelegator.MarshalPB(),
			Amount:    &someAmount,
		}
		migratedValidator := adjustValidatorIfInPlasmaValidators(migratedDelegation)
		assert.True(t, migratedValidator.Local.Compare(expectedValidator.Local) == 0)
	}

	// non-plasma validators should be left as they are
	someValidator := loom.MustParseAddress("default:0xDc93E46f6d22D47De9D7E6d26ce8c312314589Cb")
	migratedDelegationWithoutChange := Delegation{
		Validator: someValidator.MarshalPB(),
		Delegator: someDelegator.MarshalPB(),
		Amount:    &someAmount,
	}
	migratedValidator := adjustValidatorIfInPlasmaValidators(migratedDelegationWithoutChange)
	assert.True(t, migratedValidator.Local.Compare(someValidator.Local) == 0)
}

// On line: https://github.com/loomnetwork/loomchain/blob/e70bf8c95bd262dc0f2f838682756c2a1e184696/builtin/plugins/dposv2/dpos.go#L2364,
// the function creates indices for storing index counter
// with a unique key combined of validator and delegator addresses.
// If the validator of delegation is plasma validator, it will set
// validator to `plasma-0`. This causes the issue. If a user has multiple
// plasma-x delegations, he or she will end up having multiple
// plasma-0 delegations with the same index 1 during the migration
// and when the `initializationState` is set, some delegations
// will get lost.
func TestPlasmaDuplicatesLostBug(t *testing.T) {
	pctx := plugin.CreateFakeContext(addr1, addr1)

	// Deploy the coin contract (DPOS Init() will attempt to resolve it)
	coinContract := &coin.Coin{}
	coinAddr := pctx.CreateContract(coin.Contract)
	coinCtx := pctx.WithAddress(coinAddr)
	coinContract.Init(contractpb.WrapPluginContext(coinCtx), &coin.InitRequest{
		Accounts: []*coin.InitialAccount{
			makeAccount(delegatorAddress1, 1000000000000000000),
			makeAccount(PlasmaValidators[0], 1000000000000000000),
			makeAccount(PlasmaValidators[1], 1000000000000000000),
			makeAccount(PlasmaValidators[2], 1000000000000000000),
			makeAccount(PlasmaValidators[3], 1000000000000000000),
			makeAccount(PlasmaValidators[4], 1000000000000000000),
			makeAccount(PlasmaValidators[5], 1000000000000000000),
		},
	})

	registrationFee := loom.BigZeroPB()

	dposContract := &DPOS{}
	dposAddr := pctx.CreateContract(contractpb.MakePluginContract(dposContract))
	dposCtx := pctx.WithAddress(dposAddr)
	err := dposContract.Init(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &InitRequest{
		Params: &Params{
			ValidatorCount:          21,
			RegistrationRequirement: registrationFee,
		},
	})
	require.NoError(t, err)

	// Registering 5 plasma candidates
	for i, _ := range PlasmaValidators {
		err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(PlasmaValidators[i])), &coin.ApproveRequest{
			Spender: dposAddr.MarshalPB(),
			Amount:  registrationFee,
		})

		err = dposContract.RegisterCandidate2(contractpb.WrapPluginContext(dposCtx.WithSender(PlasmaValidators[i])), &RegisterCandidateRequest{
			PubKey: PlasmaPubKeys[i],
		})
		require.Nil(t, err)
	}

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// We'll make 1 delegation to each of the validators
	delegationAmount := loom.NewBigUIntFromInt(10000000)
	totalAmount := loom.NewBigUIntFromInt(60000000)
	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *totalAmount},
	})

	for i, _ := range PlasmaValidators {
		err = dposContract.Delegate2(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
			ValidatorAddress: PlasmaValidators[i].MarshalPB(),
			Amount:           &types.BigUInt{Value: *delegationAmount},
		})
		require.Nil(t, err)
	}

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Then we'll inspect the post initialization state and notice
	// that we have multiple delegations with the same index
	data, err := dposContract.ViewStateDump(
		contractpb.WrapPluginContext(dposCtx),
		&ViewStateDumpRequest{},
	)

	newDelegations := data.NewState.Delegations
	for i, _ := range newDelegations {
		localVal := loom.LocalAddressFromPublicKey(PlasmaPubKeys[0])
		assert.True(t, newDelegations[i].Validator.Local.Compare(localVal) == 0)
		// THIS is the bug. If it were implemented properly,
		// we'd have all delegations to plasma-0, but their index would be 1-7
		// instead of all 1's. When the migration executes, only 1 is applied
		// since they all have the same index.
		assert.Equal(t, newDelegations[i].Index, uint64(1))
	}
}

// UTILITIES

func makeAccount(owner loom.Address, bal uint64) *coin.InitialAccount {
	return &coin.InitialAccount{
		Owner:   owner.MarshalPB(),
		Balance: bal,
	}
}
