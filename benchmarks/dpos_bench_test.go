package benchmarks

import (
	"encoding/hex"
	"testing"

	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
	dpos "github.com/loomnetwork/loomchain/builtin/plugins/dposv3"
	loom "github.com/loomnetwork/go-loom"
	dtypes "github.com/loomnetwork/go-loom/builtin/types/dposv3"

	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/tendermint/tendermint/libs/db"
	"github.com/stretchr/testify/require"
	types "github.com/loomnetwork/go-loom/types"
)

var (
	validatorPubKeyHex1 = "3866f776276246e4f9998aa90632931d89b0d3a5930e804e02299533f55b39e1"
	validatorPubKeyHex2 = "7796b813617b283f81ea1747fbddbe73fe4b5fce0eac0728e47de51d8e506701"
	validatorPubKeyHex3 = "e4008e26428a9bca87465e8de3a8d0e9c37a56ca619d3d6202b0567528786618"
	validatorPubKeyHex4 = "21908210428a9bca87465e8de3a8d0e9c37a56ca619d3d6202b0567528786618"

	delegatorAddress1 = loom.MustParseAddress("default:0xb16a379ec18d4093666f8f38b11a3071c920207d")
	delegatorAddress2 = loom.MustParseAddress("default:0xfa4c7920accfd66b86f5fd0e69682a79f762d49e")
	delegatorAddress3 = loom.MustParseAddress("default:0x5cecd1f7261e1f4c684e297be3edf03b825e01c4")
	delegatorAddress4 = loom.MustParseAddress("default:0x000000000000000000000000e3edf03b825e01e0")
	delegatorAddress5 = loom.MustParseAddress("default:0x020000000000000000000000e3edf03b825e0288")
	delegatorAddress6 = loom.MustParseAddress("default:0x000000000000000000040400e3edf03b825e0398")

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

)

func BenchmarkLargeElection(b *testing.B) {
	for i := 0; i < b.N; i++ {
		dposInit := dtypes.DPOSInitRequest{
			Params: &dtypes.Params{
				ValidatorCount: 21,
				OracleAddress:  addr1.MarshalPB(),
			},
		}
		coinInit := coin.InitRequest{
			Accounts: []*coin.InitialAccount{
				dpos.MakeAccount(addr1, 2000000000000000000),
				dpos.MakeAccount(addr2, 2000000000000000000),
				dpos.MakeAccount(addr3, 2000000000000000000),
			},
		}

		appDbName := "dbs/mockDposDB"
		appDb, err := db.NewGoLevelDB(appDbName, ".")

		state, reg, pluginVm, err := MockStateWithDposAndCoin(&dposInit, &coinInit, appDb)
		require.NoError(b, err)

		dposContract := &dpos.DPOS{}
		dposAddr, err := reg.Resolve("dposV3")
		require.NoError(b, err)

		/*
		dposTestContract := &dpos.TestDPOSContract{
			Contract: dposContract,
			Address: dposAddr,
		}
		*/

		dposCtx := contractpb.WrapPluginContext(
			CreateFakeStateContext(state, reg, addr1, dposAddr, pluginVm),
		)

		coinAddr, err := reg.Resolve("coin")
		require.NoError(b, err)
		coinContract := &coin.Coin{}
		coinCtx := contractpb.WrapPluginContext(
			CreateFakeStateContext(state, reg, addr1, coinAddr, pluginVm),
		)

		require.NoError(b, coinContract.Approve(coinCtx, &coin.ApproveRequest{
			Spender: dposAddr.MarshalPB(),
			Amount:  &types.BigUInt{Value: *loom.NewBigUIntFromInt(100000000000000000)},
		}))

		err = dposContract.RegisterCandidate(dposCtx, &dtypes.RegisterCandidateRequest{
			PubKey: pubKey1,
		})
		require.Nil(b, err)
		/*
		err = dposTestContract.RegisterCandidate(dposCtx.WithSender(addr1), pubKey1, nil, nil, nil, nil, nil, nil)
		require.Nil(b, err)
		err = dposTestContract.RegisterCandidate(dposCtx.WithSender(addr2), pubKey2, nil, nil, nil, nil, nil, nil)
		require.Nil(b, err)
		err = dposTestContract.RegisterCandidate(dposCtx.WithSender(addr3), pubKey3, nil, nil, nil, nil, nil, nil)
		require.Nil(b, err)
		*/
	}
}
