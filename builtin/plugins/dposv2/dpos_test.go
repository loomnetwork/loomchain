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

	d2types "github.com/loomnetwork/go-loom/builtin/types/dposv2"
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

	err = dposContract.RegisterCandidate(contractpb.WrapPluginContext(pctx.WithSender(addr)), &RegisterCandidateRequest{
		PubKey: pubKey,
	})
	require.Nil(t, err)

	err = dposContract.UnregisterCandidate(contractpb.WrapPluginContext(pctx.WithSender(addr)), &UnregisterCandidateRequest{})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate(contractpb.WrapPluginContext(pctx.WithSender(addr)), &RegisterCandidateRequest{
		PubKey: pubKey,
	})
	require.Nil(t, err)

	err = dposContract.RemoveWhitelistedCandidate(contractpb.WrapPluginContext(pctx.WithSender(oracleAddr)), &RemoveWhitelistedCandidateRequest{
		CandidateAddress: addr.MarshalPB(),
	})
	require.Nil(t, err)

	err = dposContract.UnregisterCandidate(contractpb.WrapPluginContext(pctx.WithSender(addr)), &UnregisterCandidateRequest{})
	require.Nil(t, err)
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
			makeAccount(addr1, 2000000000000000000),
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

	err = dposContract.RegisterCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	// Candidate registered, let's make a delegation to them

	approvalAmount := &types.BigUInt{Value: loom.BigUInt{big.NewInt(300)}}
	delegationAmount := &types.BigUInt{Value: loom.BigUInt{big.NewInt(100)}}
	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  approvalAmount,
	})
	require.Nil(t, err)

	bigLockTime := uint64(15780000) // 6 months in seconds
	err = dposContract.Delegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
		LockTime:         bigLockTime,
	})
	require.Nil(t, err)

	delegation1Response, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)

	d1LockTime := delegation1Response.Delegation.LockTime
	assert.Equal(t, true, d1LockTime == bigLockTime)

	// Elections must happen so that we delegate again
	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Try delegating with a LockTime set to be less. It won't matter, since the existing locktime will be used.
	smallerLockTime := bigLockTime / 2 // 3 months in seconds
	now := uint64(dposCtx.Now().Unix())
	err = dposContract.Delegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
		LockTime:         smallerLockTime,
	})
	require.Nil(t, err)

	delegation2Response, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	d2LockTime := delegation2Response.Delegation.LockTime
	// New locktime should be the `now` value extended by the previous locktime
	assert.Equal(t, d2LockTime, now+d1LockTime)


	// Elections must happen so that we delegate again
	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	// Try delegating with a LockTime set to be bigger. It will overwrite the old locktime.
	biggerLockTime := bigLockTime * 2 // 12 months in seconds
	now = uint64(dposCtx.Now().Unix())
	err = dposContract.Delegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
		LockTime:         biggerLockTime,
	})
	require.Nil(t, err)

	delegation3Response, err := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &CheckDelegationRequest{
		ValidatorAddress: addr1.MarshalPB(),
		DelegatorAddress: delegatorAddress1.MarshalPB(),
	})
	require.Nil(t, err)
	d3LockTime := delegation3Response.Delegation.LockTime

	// New locktime should be the `now` value extended by the new locktime
	assert.Equal(t, d3LockTime, now+d3LockTime)
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

	err = dposContract.RegisterCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
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
	err = dposContract.Delegate(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  delegationAmount,
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposCtx))
	require.Nil(t, err)

	err = dposContract.Delegate(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           delegationAmount,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  delegationAmount,
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
	require.Nil(t, err)

	err = dposContract.ProcessRequestBatch(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RequestBatch{
		Batch: []*d2types.BatchRequestV2{
			&d2types.BatchRequestV2{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr2.MarshalPB(),
					Amount:           &types.BigUInt{Value: loom.BigUInt{big.NewInt(1000000000000)}},
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
					Amount:           &types.BigUInt{Value: loom.BigUInt{big.NewInt(1000000000000)}},
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

	err = dposContract.RegisterCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(addr3)), &RegisterCandidateRequest{
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

	registrationFee := &types.BigUInt{Value: *scientificNotation(registrationRequirement, tokenDecimals)}

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr1)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr2)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(addr2)), &RegisterCandidateRequest{
		PubKey: pubKey2,
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(addr3)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  registrationFee,
	})
	require.Nil(t, err)

	err = dposContract.RegisterCandidate(contractpb.WrapPluginContext(dposCtx.WithSender(addr3)), &RegisterCandidateRequest{
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

	err = dposContract.Delegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress1)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *smallDelegationAmount},
	})
	require.Nil(t, err)

	err = coinContract.Approve(contractpb.WrapPluginContext(coinCtx.WithSender(delegatorAddress2)), &coin.ApproveRequest{
		Spender: dposAddr.MarshalPB(),
		Amount:  &types.BigUInt{Value: *largeDelegationAmount},
	})
	require.Nil(t, err)

	err = dposContract.Delegate(contractpb.WrapPluginContext(dposCtx.WithSender(delegatorAddress2)), &DelegateRequest{
		ValidatorAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: *largeDelegationAmount},
	})
	require.Nil(t, err)

	for i := 0; i < 1000; i = i + 1 {
		err = Elect(contractpb.WrapPluginContext(dposCtx))
		require.Nil(t, err)
	}

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

	halvedDelegator2Claim := loom.NewBigUIntFromInt(0)
	halvedDelegator2Claim.Div(&delegator2Claim.Amount.Value, loom.NewBigUIntFromInt(2))
	difference := loom.NewBigUIntFromInt(0)
	difference.Sub(&delegator1Claim.Amount.Value, halvedDelegator2Claim)

	// Checking that Delegator2's claim is almost exactly half of Delegator1's claim
	maximumDifference := scientificNotation(1, tokenDecimals)
	assert.Equal(t, difference.Int.CmpAbs(maximumDifference.Int), -1)
}

// UTILITIES

func makeAccount(owner loom.Address, bal uint64) *coin.InitialAccount {
	return &coin.InitialAccount{
		Owner:   owner.MarshalPB(),
		Balance: bal,
	}
}
