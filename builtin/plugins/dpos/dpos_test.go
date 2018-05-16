package main

import (
	"encoding/hex"
	"testing"

	loom "github.com/loomnetwork/go-loom"
	types "github.com/loomnetwork/go-loom/builtin/types/dpos"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/stretchr/testify/require"
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
	pubKey, _ := hex.DecodeString(valPubKeyHex1)
	addr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey),
	}
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr, addr),
	)
	c := &DPOS{}

	err := c.RegisterCandidate(ctx, &types.RegisterCandidateRequest{
		PubKey: pubKey,
	})
	require.Nil(t, err)
}

func TestVote(t *testing.T) {
	pubKey1, _ := hex.DecodeString(valPubKeyHex1)
	addr1 := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey1),
	}
	pctx := plugin.CreateFakeContext(addr1, addr1)
	ctx := contractpb.WrapPluginContext(pctx)
	c := &DPOS{}
	err := c.Init(ctx, &types.InitRequest{
		Params: &types.Params{
			ValidatorCount: 21,
		},
	})
	require.Nil(t, err)

	err = c.RegisterCandidate(ctx, &types.RegisterCandidateRequest{
		PubKey: pubKey1,
	})
	require.Nil(t, err)

	// Too many votes given
	ctx = contractpb.WrapPluginContext(pctx.WithSender(voterAddr1))
	err = c.Vote(ctx, &types.VoteRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           50,
	})
	require.NotNil(t, err)

	err = c.Vote(ctx, &types.VoteRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           20,
	})
	require.Nil(t, err)

	err = c.Vote(ctx, &types.VoteRequest{
		CandidateAddress: addr1.MarshalPB(),
		Amount:           2,
	})
	require.NotNil(t, err)
}
