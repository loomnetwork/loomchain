package dposv2

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/loomnetwork/go-loom"
	common "github.com/loomnetwork/go-loom/common"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	types "github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"

	"github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	d2types "github.com/loomnetwork/go-loom/builtin/types/dposv2"
)

// TODO test the situation where there are redundant delegations in dposv2

func TestMigration(t *testing.T) {
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

	// create dposv2 contract
	dposv2Contract := &DPOS{}
	dposv2Addr := pctx.CreateContract(contractpb.MakePluginContract(dposv2Contract))
	dposv2Ctx := pctx.WithAddress(dposv2Addr)

	// create dposv3 contract
	dposv3Contract := &dposv3.DPOS{}
	dposv3Addr := pctx.CreateContract(contractpb.MakePluginContract(dposv3Contract))
	// dposv3Ctx := pctx.WithAddress(dposv3Addr)

	// transfer coins to reward fund
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(19), nil)
	coinContract.Transfer(contractpb.WrapPluginContext(coinCtx), &coin.TransferRequest{
		To: dposv3Addr.MarshalPB(),
		Amount: &types.BigUInt{
			Value: common.BigUInt{amount},
		},
	})

	// Init the dpos contract
	err := dposv2Contract.Init(contractpb.WrapPluginContext(dposv2Ctx.WithSender(addr1)), &InitRequest{
		Params: &Params{
			CoinContractAddress: coinAddr.MarshalPB(),
			ValidatorCount:      2,
			ElectionCycleLength: 0,
			OracleAddress:       addr1.MarshalPB(),
		},
	})
	require.Nil(t, err)

	whitelistAmount := loom.BigUInt{big.NewInt(1000000000000)}

	err = dposv2Contract.ProcessRequestBatch(contractpb.WrapPluginContext(dposv2Ctx.WithSender(addr1)), &RequestBatch{
		Batch: []*BatchRequest{
			&BatchRequest{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr1.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         10,
				}},
				Meta: &BatchRequestMeta{
					BlockNumber: 1,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = dposv2Contract.ProcessRequestBatch(contractpb.WrapPluginContext(dposv2Ctx.WithSender(addr1)), &RequestBatch{
		Batch: []*BatchRequest{
			&BatchRequest{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr2.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         10,
				}},
				Meta: &BatchRequestMeta{
					BlockNumber: 2,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = dposv2Contract.ProcessRequestBatch(contractpb.WrapPluginContext(dposv2Ctx.WithSender(addr1)), &RequestBatch{
		Batch: []*BatchRequest{
			&BatchRequest{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&WhitelistCandidateRequest{
					CandidateAddress: addr3.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         10,
				}},
				Meta: &BatchRequestMeta{
					BlockNumber: 3,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	/*
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

	oldRewardsValue := *common.BigZero()
	for i := 0; i < 10; i++ {
		err = Elect(contractpb.WrapPluginContext(dposCtx))
		require.Nil(t, err)
		checkDelegation, _ := dposContract.CheckDelegation(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &CheckDelegationRequest{
			ValidatorAddress: addr1.MarshalPB(),
			DelegatorAddress: addr1.MarshalPB(),
		})
		// get rewards delegaiton which is always at index 0
		delegation := checkDelegation.Delegations[REWARD_DELEGATION_INDEX]
		assert.Equal(t, delegation.Amount.Value.Cmp(&oldRewardsValue), 1)
		oldRewardsValue = delegation.Amount.Value
	}

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
	require.Error(t, err)

	err = dposContract.ChangeWhitelistAmount(contractpb.WrapPluginContext(dposCtx.WithSender(addr1)), &ChangeWhitelistAmountRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           &types.BigUInt{Value: newWhitelistAmount},
	})
	require.Nil(t, err)

	listValidatorsResponse, err = dposContract.ListValidators(contractpb.WrapPluginContext(dposCtx), &ListValidatorsRequest{})
	require.Nil(t, err)
	validator = listValidatorsResponse.Statistics[0]
	assert.Equal(t, newWhitelistAmount, validator.WhitelistAmount.Value)
	*/
}
