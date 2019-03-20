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
	dposv3Ctx := pctx.WithAddress(dposv3Addr)

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

	err = dposv2Contract.RegisterCandidate(contractpb.WrapPluginContext(dposv2Ctx.WithSender(addr1)), &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	err = Elect(contractpb.WrapPluginContext(dposv2Ctx))
	require.Nil(t, err)

	// 
	err = Elect(contractpb.WrapPluginContext(dposv2Ctx))
	require.Nil(t, err)

	err = dposv2Contract.Dump(contractpb.WrapPluginContext(dposv2Ctx), dposv3Addr)
	require.Nil(t, err)

	listValidatorsResponse, err := dposv3Contract.ListValidators(contractpb.WrapPluginContext(dposv3Ctx), &dposv3.ListValidatorsRequest{})
	require.Nil(t, err)
	assert.Equal(t, len(listValidatorsResponse.Statistics), 1)
	validator := listValidatorsResponse.Statistics[0]
	assert.Equal(t, whitelistAmount, validator.WhitelistAmount.Value)
}
