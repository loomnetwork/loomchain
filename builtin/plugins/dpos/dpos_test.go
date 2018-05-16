package dpos

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/types"
	"github.com/loomnetwork/loomchain/builtin/plugins/coin"
)

var (
	valPubKeyHex1 = "3866f776276246e4f9998aa90632931d89b0d3a5930e804e02299533f55b39e1"
	valPubKeyHex2 = "7796b813617b283f81ea1747fbddbe73fe4b5fce0eac0728e47de51d8e506701"
	valPubKeyHex3 = "e4008e26428a9bca87465e8de3a8d0e9c37a56ca619d3d6202b0567528786618"

	voterAddr1 = loom.MustParseAddress("chain:b16a379ec18d4093666f8f38b11a3071c920207d")
	voterAddr2 = loom.MustParseAddress("chain:fa4c7920accfd66b86f5fd0e69682a79f762d49e")
	voterAddr3 = loom.MustParseAddress("chain:5cecd1f7261e1f4c684e297be3edf03b825e01c4")
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

func TestVote(t *testing.T) {
	c := &DPOS{}

	pubKey1, _ := hex.DecodeString(valPubKeyHex1)
	addr1 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey1),
	}
	pctx := plugin.CreateFakeContext(addr1, addr1)
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

	// Too many votes given
	ctx = contractpb.WrapPluginContext(pctx.WithSender(voterAddr1))
	err = c.Vote(ctx, &VoteRequest{
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
}

func makeAccount(owner loom.Address, bal uint64) *coin.Account {
	val := loom.NewBigUIntFromInt(10)
	val.Exp(val, loom.NewBigUIntFromInt(18), nil)
	val.Mul(val, loom.NewBigUIntFromInt(int64(bal)))

	return &coin.Account{
		Owner: owner.MarshalPB(),
		Balance: &types.BigUInt{
			Value: *val,
		},
	}
}

func TestElect(t *testing.T) {
	pctx := plugin.CreateFakeContext(voterAddr1, loom.Address{})
	coinAddr := pctx.CreateContract(coin.Contract)

	coinContract := &coin.Coin{}
	ctx := contractpb.WrapPluginContext(pctx.WithAddress(coinAddr))
	coinContract.Init(ctx, &coin.InitRequest{
		Accounts: []*coin.Account{
			makeAccount(voterAddr1, 10),
			makeAccount(voterAddr2, 20),
			makeAccount(voterAddr3, 30),
		},
	})
	c := &DPOS{}

	pubKey1, _ := hex.DecodeString(valPubKeyHex1)
	addr1 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey1),
	}

	// switch to dpos contract context
	pctx = pctx.WithAddress(loom.Address{})

	ctx = contractpb.WrapPluginContext(pctx.WithSender(addr1))
	err := c.Init(ctx, &InitRequest{
		Params: &Params{
			CoinContractAddress: coinAddr.MarshalPB(),
			ValidatorCount:      21,
		},
	})
	require.Nil(t, err)

	err = c.RegisterCandidate(ctx, &RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	ctx = contractpb.WrapPluginContext(pctx.WithSender(voterAddr1))

	err = c.Vote(ctx, &VoteRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           20,
	})
	require.Nil(t, err)

	ctx = contractpb.WrapPluginContext(pctx.WithSender(voterAddr2))

	err = c.Vote(ctx, &VoteRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           20,
	})
	require.Nil(t, err)

	err = c.Elect(ctx, &ElectRequest{})
	require.Nil(t, err)
}
