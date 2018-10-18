package dposv2

import (
	"encoding/hex"
	"testing"

	loom "github.com/loomnetwork/go-loom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustAddrFromPubKey(s string) loom.Address {
	pubKey, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey),
	}
}

func TestElection(t *testing.T) {
	canAddr1 := mustAddrFromPubKey(valPubKeyHex1)
	canAddr2 := mustAddrFromPubKey(valPubKeyHex2)
	canAddr3 := mustAddrFromPubKey(valPubKeyHex3)

	votes := []*FullVote{
		&FullVote{
			CandidateAddress: canAddr1,
			VoteSize:         80,
			Power:            100,
		},
		&FullVote{
			CandidateAddress: canAddr2,
			VoteSize:         20,
			Power:            300,
		},
		&FullVote{
			CandidateAddress: canAddr3,
			VoteSize:         20,
			Power:            300,
		},
		&FullVote{
			CandidateAddress: canAddr2,
			VoteSize:         5,
			Power:            100,
		},
	}

	results, err := runElection(votes)
	require.Nil(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, 400, int(results[0].PowerTotal))
	assert.Equal(t, canAddr2, results[0].CandidateAddress)
	assert.Equal(t, 100, int(results[2].PowerTotal))
	assert.Equal(t, canAddr1, results[2].CandidateAddress)
}
