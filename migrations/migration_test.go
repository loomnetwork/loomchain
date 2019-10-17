package migrations

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

	d2types "github.com/loomnetwork/go-loom/builtin/types/dposv2"
	"github.com/loomnetwork/loomchain/builtin/plugins/dposv2"
)

var (
	validatorPubKeyHex1 = "3866f776276246e4f9998aa90632931d89b0d3a5930e804e02299533f55b39e1"

	delegatorAddress1 = loom.MustParseAddress("chain:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	delegatorAddress2 = loom.MustParseAddress("chain:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	delegatorAddress3 = loom.MustParseAddress("chain:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
)

// TODO test the situation where there are redundant delegations in dposv2

func TestMigration(t *testing.T) {
	chainID := "chain"
	pubKey1, _ := hex.DecodeString(validatorPubKeyHex1)
	addr1 := loom.Address{
		ChainID: chainID,
		Local:   loom.LocalAddressFromPublicKey(pubKey1),
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
	dposv2Contract := &dposv2.DPOS{}
	dposv2Addr := pctx.CreateContract(contractpb.MakePluginContract(dposv2Contract))
	dposv2Ctx := pctx.WithAddress(dposv2Addr)

	// transfer coins to reward fund
	amount := big.NewInt(10)
	amount.Exp(amount, big.NewInt(19), nil)
	coinContract.Transfer(contractpb.WrapPluginContext(coinCtx), &coin.TransferRequest{
		To: dposv2Addr.MarshalPB(),
		Amount: &types.BigUInt{
			Value: common.BigUInt{amount},
		},
	})

	// Init the dpos contract
	err := dposv2Contract.Init(contractpb.WrapPluginContext(dposv2Ctx.WithSender(addr1)), &dposv2.InitRequest{
		Params: &dposv2.Params{
			CoinContractAddress: coinAddr.MarshalPB(),
			ValidatorCount:      2,
			ElectionCycleLength: 0,
			OracleAddress:       addr1.MarshalPB(),
		},
	})
	require.Nil(t, err)

	whitelistAmount := loom.BigUInt{big.NewInt(1000000000000)}

	err = dposv2Contract.ProcessRequestBatch(contractpb.WrapPluginContext(dposv2Ctx.WithSender(addr1)), &dposv2.RequestBatch{
		Batch: []*dposv2.BatchRequest{
			{
				Payload: &d2types.BatchRequestV2_WhitelistCandidate{&dposv2.WhitelistCandidateRequest{
					CandidateAddress: addr1.MarshalPB(),
					Amount:           &types.BigUInt{Value: whitelistAmount},
					LockTime:         10,
				}},
				Meta: &dposv2.BatchRequestMeta{
					BlockNumber: 1,
					TxIndex:     0,
					LogIndex:    0,
				},
			},
		},
	})
	require.Nil(t, err)

	err = dposv2Contract.RegisterCandidate(contractpb.WrapPluginContext(dposv2Ctx.WithSender(addr1)), &dposv2.RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	err = dposv2.Elect(contractpb.WrapPluginContext(dposv2Ctx))
	require.Nil(t, err)

	// running a second election to make sure addr1 gets a reward delegation
	err = dposv2.Elect(contractpb.WrapPluginContext(dposv2Ctx))
	require.Nil(t, err)

	// DPOSv3Migration(dposv2Ctx)

	// // create dposv3 contract
	// dposv3Contract := &dposv3.DPOS{}
	// dposv3Addr := pctx.CreateContract(contractpb.MakePluginContract(dposv3Contract))
	// // dposv3Ctx := pctx.WithAddress(dposv3Addr)

	// listValidatorsResponse, err := dposv3Contract.ListValidators(contractpb.WrapPluginContext(dposv3Ctx), &dposv3.ListValidatorsRequest{})
	// require.Nil(t, err)
	// assert.Equal(t, len(listValidatorsResponse.Statistics), 1)
	// validator := listValidatorsResponse.Statistics[0]
	// assert.Equal(t, whitelistAmount, validator.WhitelistAmount.Value)
}

// UTILITIES

func makeAccount(owner loom.Address, bal uint64) *coin.InitialAccount {
	return &coin.InitialAccount{
		Owner:   owner.MarshalPB(),
		Balance: bal,
	}
}
